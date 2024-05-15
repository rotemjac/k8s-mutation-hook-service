// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package mutate

import (
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// Mutate mutates
func Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := &admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	var pod *corev1.Pod

	responseBody := []byte{}
	admReviewRequest := admReview.Request
	admResponse := v1.AdmissionResponse{}

	if admReviewRequest != nil {
		// Get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(admReviewRequest.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}

		// The actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
		// tell K8S how it should modifiy it
		//p := []map[string]string{}

		//var patchStr string

		currentTime := time.Now()
		timestamp := currentTime.Format("2006.01.02 15:04:05")
		timestamp = strings.Replace(timestamp, ".", "-", -1)
		timestamp = strings.Replace(timestamp, ":", "-", -1)
		timestamp = strings.Replace(timestamp, " ", "-", -1)

		//path := "/metadata/annotations"
		prefix := pod.Annotations["dd.replace/prefix"]
		//postfix := pod.Annotations["dd.replace/postfix"]
		podName := pod.GetGenerateName() + "-" + timestamp
		annotationKey := fmt.Sprintf(prefix + "/" + podName)
		var patch []PatchOperation

		//ad.datadoghq.com/trino-worker-<podName>.check_names
		key1 := annotationKey + ".check_names"
		value1 := pod.Annotations["dd.replace/check_names"]

		//ad.datadoghq.com/trino-worker-<podName>.init_configs
		key2 := annotationKey + ".init_configs"
		value2 := "'[{}]'"

		//ad.datadoghq.com/trino-worker-<podName>.instances
		key3 := annotationKey + ".instances"
		value3 := pod.Annotations["dd.replace/instances"]
		value3 = strings.Replace(value3, "trino-worker-", podName, -1)

		patch = append(patch, PatchOperation{
			Op:   "add",
			Path: "/metadata/annotations",
			Value: map[string]string{
				key1: value1,
				key2: value2,
				key3: value3,
			},
		})

		// #######################################
		pT := v1.PatchTypeJSONPatch
		admResponse.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// Parse the []map into JSON
		admResponse.Patch, err = json.Marshal(patch)

		// Set response options
		admReview.Response = &admResponse
		admResponse.Allowed = true

		// Construct the response, which is just another AdmissionReview.
		var admissionReviewResponse v1.AdmissionReview
		admissionReviewResponse.TypeMeta = metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"}
		admissionReviewResponse.Response = &admResponse
		//admissionReviewResponse.SetGroupVersionKind(admReviewRequest.GroupVersionKind())
		admissionReviewResponse.Response.UID = admReviewRequest.UID

		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admissionReviewResponse)
		if err != nil {
			return nil, err // untested section
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
