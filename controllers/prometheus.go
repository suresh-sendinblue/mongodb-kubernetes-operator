package controllers

import (
	"fmt"

	mdbv1 "github.com/mongodb/mongodb-kubernetes-operator/api/v1"
	"github.com/mongodb/mongodb-kubernetes-operator/pkg/automationconfig"
	"github.com/mongodb/mongodb-kubernetes-operator/pkg/kube/secret"
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/types"
)

const (
	listenAddress = "0.0.0.0"
)

// PrometheusModification adds Prometheus configuration to AutomationConfig.
func getPrometheusModification(getUpdateCreator secret.GetUpdateCreator, mdb mdbv1.MongoDBCommunity) (automationconfig.Modification, error) {
	if mdb.Spec.Prometheus == nil {
		return automationconfig.NOOP(), nil
	}

	secretNamespacedName := types.NamespacedName{Name: mdb.Spec.Prometheus.PasswordSecretRef.Name, Namespace: mdb.Namespace}
	password, err := secret.ReadKey(getUpdateCreator, mdb.Spec.Prometheus.GetPasswordKey(), secretNamespacedName)
	if err != nil {
		return automationconfig.NOOP(), errors.Errorf("could not configure Prometheus modification: %s", err)
	}

	var certKey string
	var tlsPEMPath string
	var scheme string

	if mdb.Spec.Prometheus.TLSSecretRef.Name != "" {
		certKey, err = getPemOrConcatenatedCrtAndKey(getUpdateCreator, mdb, mdb.PrometheusTLSSecretNamespacedName())
		if err != nil {
			return automationconfig.NOOP(), err
		}
		tlsPEMPath = tlsOperatorSecretMountPath + tlsOperatorSecretFileName(certKey)
		scheme = "https"
	} else {
		scheme = "http"
	}

	return func(config *automationconfig.AutomationConfig) {
		promConfig := automationconfig.NewDefaultPrometheus(mdb.Spec.Prometheus.Username)

		promConfig.TLSPemPath = tlsPEMPath
		promConfig.Scheme = scheme
		promConfig.Password = password

		if mdb.Spec.Prometheus.Port > 0 {
			promConfig.ListenAddress = fmt.Sprintf("%s:%d", listenAddress, mdb.Spec.Prometheus.Port)
		}

		if mdb.Spec.Prometheus.MetricsPath != "" {
			promConfig.MetricsPath = mdb.Spec.Prometheus.MetricsPath
		}

		config.Prometheus = &promConfig
	}, nil
}
