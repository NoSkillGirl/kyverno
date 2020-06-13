package webhooks

import (
	"fmt"

	"github.com/jimlawless/whereami"
	"k8s.io/api/admission/v1beta1"
)

func (ws *WebhookServer) verifyHandler(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	fmt.Printf("%s\n", whereami.WhereAmI())
	logger := ws.log.WithValues("action", "verify", "uid", request.UID, "kind", request.Kind, "namespace", request.Namespace, "name", request.Name, "operation", request.Operation)
	logger.V(4).Info("incoming request")
	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}
