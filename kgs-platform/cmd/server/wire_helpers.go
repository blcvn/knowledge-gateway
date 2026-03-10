package main

import (
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
)

func newOntologyValidatorConfig(confData *conf.Data) biz.OntologyValidatorConfig {
	ontology := confData.GetOntology()
	if ontology == nil {
		return biz.OntologyValidatorConfig{}
	}
	return biz.OntologyValidatorConfig{
		Enabled:             ontology.GetValidationEnabled(),
		StrictMode:          ontology.GetStrictMode(),
		SchemaValidation:    ontology.GetSchemaValidation(),
		EdgeConstraintCheck: ontology.GetEdgeConstraintCheck(),
	}
}
