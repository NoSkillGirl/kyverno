package policy

import (
	"fmt"

	"github.com/jimlawless/whereami"
	"k8s.io/apimachinery/pkg/labels"
)

func (pc *PolicyController) removeResourceWebhookConfiguration() error {
	fmt.Printf("%s\n", whereami.WhereAmI())
	logger := pc.log
	var err error
	// get all existing policies
	policies, err := pc.pLister.List(labels.NewSelector())
	if err != nil {
		logger.Error(err, "failed to list policies")
		return err
	}

	if len(policies) == 0 {
		logger.V(4).Info("no policies loaded, removing resource webhook configuration if one exists")
		return pc.resourceWebhookWatcher.RemoveResourceWebhookConfiguration()
	}

	logger.V(4).Info("no policies with mutating or validating webhook configurations, remove resource webhook configuration if one exists")

	return pc.resourceWebhookWatcher.RemoveResourceWebhookConfiguration()
}

func (pc *PolicyController) registerResourceWebhookConfiguration() {
	fmt.Printf("%s\n", whereami.WhereAmI())
	pc.resourceWebhookWatcher.RegisterResourceWebhook()
}
