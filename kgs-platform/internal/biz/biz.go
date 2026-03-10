package biz

import "github.com/google/wire"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewGreeterUsecase, NewOntologySyncManager, NewOntologyValidator, NewGraphUsecase, NewQueryPlanner, NewViewResolver, NewRulesUsecase, NewPolicyUsecase, NewRuleRunner, NewOPAClient, NewEventRunner, NewPolicySyncRunner, NewRegistryUsecase)
