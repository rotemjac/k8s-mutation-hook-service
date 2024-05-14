// Package mutate deals with AdmissionReview requests and responses, it takes in the request body and returns a readily converted JSON []byte that can be
// returned from a http Handler w/o needing to further convert or modify it, it also makes testing Mutate() kind of easy w/o need for a fake http server, etc.
package mutate

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	v1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mutate mutates
func Mutate(body []byte, verbose bool) ([]byte, error) {
	if verbose {
		log.Printf("recv: %s\n", string(body)) // untested section
	}

	// unmarshal request into AdmissionReview struct
	admReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReview); err != nil {
		return nil, fmt.Errorf("unmarshaling request failed with %s", err)
	}

	var err error
	var pod *corev1.Pod

	responseBody := []byte{}
	ar := admReview.Request
	resp := v1beta1.AdmissionResponse{}

	if ar != nil {

		// get the Pod object and unmarshal it into its struct, if we cannot, we might as well stop here
		if err := json.Unmarshal(ar.Object.Raw, &pod); err != nil {
			return nil, fmt.Errorf("unable unmarshal pod json object %v", err)
		}
		// set response options
		resp.Allowed = true
		resp.UID = ar.UID
		pT := v1beta1.PatchTypeJSONPatch
		resp.PatchType = &pT // it's annoying that this needs to be a pointer as you cannot give a pointer to a constant?

		// the actual mutation is done by a string in JSONPatch style, i.e. we don't _actually_ modify the object, but
		// tell K8S how it should modifiy it
		p := []map[string]string{}

		prefix := pod.Annotations["dd.replace/prefix"]
		postfix := pod.Annotations["dd.replace/postfix"]
		annotationKey := fmt.Sprintf(prefix+"/"+postfix+"-%s", pod.Name)

		//ad.datadoghq.com/trino-worker-<podName>.check_names
		checkNamesPath := annotationKey + ".check_names"
		checkNamesValue := pod.Annotations["dd.replace/check_names"]
		patch := map[string]string{
			"op":    "add",
			"path":  checkNamesPath,
			"value": checkNamesValue,
		}
		p = append(p, patch)

		//ad.datadoghq.com/trino-worker-<podName>.init_configs
		checkNamesPath = annotationKey + ".init_configs"
		checkNamesValue = pod.Annotations["dd.replace/init_configs"]
		patch = map[string]string{
			"op":    "add",
			"path":  checkNamesPath,
			"value": checkNamesValue,
		}
		p = append(p, patch)

		//ad.datadoghq.com/trino-worker-<podName>.instances
		checkNamesPath = annotationKey + ".instances"
		checkNamesValue = pod.Annotations["dd.replace/instances"]
		checkNamesValue = strings.Replace(checkNamesValue, "trino-worker-", "trino-worker-"+pod.Name, -1)
		patch = map[string]string{
			"op":    "add",
			"path":  checkNamesPath,
			"value": checkNamesValue,
		}
		p = append(p, patch)

		// parse the []map into JSON
		resp.Patch, err = json.Marshal(p)

		// Success, of course ;)
		resp.Result = &metav1.Status{
			Status: "Success",
		}

		admReview.Response = &resp
		// back into JSON so we can return the finished AdmissionReview w/ Response directly
		// w/o needing to convert things in the http handler
		responseBody, err = json.Marshal(admReview)
		if err != nil {
			return nil, err // untested section
		}
	}

	if verbose {
		log.Printf("resp: %s\n", string(responseBody)) // untested section
	}

	return responseBody, nil
}
