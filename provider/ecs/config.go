package ecs

import (
	"math"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/ty/fun"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/types"
)

// buildConfiguration fills the config template with the given instances
func (p *Provider) buildConfiguration(services map[string][]ecsInstance) (*types.Configuration, error) {
	var ecsFuncMap = template.FuncMap{
		"filterFrontends":             filterFrontends,
		"getFrontendRule":             p.getFrontendRule,
		"getBasicAuth":                getFuncSliceString(label.TraefikFrontendAuthBasic),
		"hasLoadBalancerLabel":        hasLoadBalancerLabel,
		"getLoadBalancerMethod":       getFuncFirstStringValue(label.TraefikBackendLoadBalancerMethod, label.DefaultBackendLoadBalancerMethod),
		"getSticky":                   getSticky,
		"hasStickinessLabel":          getFuncFirstBoolValue(label.TraefikBackendLoadBalancerStickiness, false),
		"getStickinessCookieName":     getFuncFirstStringValue(label.TraefikBackendLoadBalancerStickinessCookieName, label.DefaultBackendLoadbalancerStickinessCookieName),
		"getProtocol":                 getFuncStringValue(label.TraefikProtocol, label.DefaultProtocol),
		"getHost":                     getHost,
		"getPort":                     getPort,
		"getWeight":                   getFuncStringValue(label.TraefikWeight, label.DefaultWeight),
		"getPassHostHeader":           getFuncStringValue(label.TraefikFrontendPassHostHeader, label.DefaultPassHostHeader),
		"getPassTLSCert":              getFuncBoolValue(label.TraefikFrontendPassTLSCert, label.DefaultPassTLSCert),
		"getPriority":                 getFuncStringValue(label.TraefikFrontendPriority, label.DefaultFrontendPriority),
		"getEntryPoints":              getFuncSliceString(label.TraefikFrontendEntryPoints),
		"hasHealthCheckLabels":        hasFuncFirst(label.TraefikBackendHealthCheckPath),
		"getHealthCheckPath":          getFuncFirstStringValue(label.TraefikBackendHealthCheckPath, ""),
		"getHealthCheckPort":          getFuncFirstIntValue(label.TraefikBackendHealthCheckPort, label.DefaultBackendHealthCheckPort),
		"getHealthCheckInterval":      getFuncFirstStringValue(label.TraefikBackendHealthCheckInterval, ""),
		"hasCircuitBreakerLabel":      hasFuncFirst(label.TraefikBackendCircuitBreakerExpression),
		"getCircuitBreakerExpression": getFuncFirstStringValue(label.TraefikBackendCircuitBreakerExpression, label.DefaultCircuitBreakerExpression),
		"hasMaxConnLabels":            hasMaxConnLabels,
		"getMaxConnAmount":            getFuncFirstInt64Value(label.TraefikBackendMaxConnAmount, math.MaxInt64),
		"getMaxConnExtractorFunc":     getFuncFirstStringValue(label.TraefikBackendMaxConnExtractorFunc, label.DefaultBackendMaxconnExtractorFunc),
	}
	return p.GetConfiguration("templates/ecs.tmpl", ecsFuncMap, struct {
		Services map[string][]ecsInstance
	}{
		services,
	})
}

func (p *Provider) getFrontendRule(i ecsInstance) string {
	defaultRule := "Host:" + strings.ToLower(strings.Replace(i.Name, "_", "-", -1)) + "." + p.Domain
	return getStringValue(i, label.TraefikFrontendRule, defaultRule)
}

// TODO: Deprecated
// Deprecated replaced by Stickiness
func getSticky(instances []ecsInstance) string {
	if hasFirst(instances, label.TraefikBackendLoadBalancerSticky) {
		log.Warnf("Deprecated configuration found: %s. Please use %s.", label.TraefikBackendLoadBalancerSticky, label.TraefikBackendLoadBalancerStickiness)
	}
	return getFirstStringValue(instances, label.TraefikBackendLoadBalancerSticky, "false")
}

func getHost(i ecsInstance) string {
	return *i.machine.PrivateIpAddress
}

func getPort(i ecsInstance) string {
	if value := getStringValue(i, label.TraefikPort, ""); len(value) > 0 {
		return value
	}
	return strconv.FormatInt(*i.container.NetworkBindings[0].HostPort, 10)
}

func filterFrontends(instances []ecsInstance) []ecsInstance {
	byName := make(map[string]struct{})

	return fun.Filter(func(i ecsInstance) bool {
		_, found := byName[i.Name]
		if !found {
			byName[i.Name] = struct{}{}
		}
		return !found
	}, instances).([]ecsInstance)
}

func hasLoadBalancerLabel(instances []ecsInstance) bool {
	method := hasFirst(instances, label.TraefikBackendLoadBalancerMethod)
	sticky := hasFirst(instances, label.TraefikBackendLoadBalancerSticky)
	stickiness := hasFirst(instances, label.TraefikBackendLoadBalancerStickiness)
	cookieName := hasFirst(instances, label.TraefikBackendLoadBalancerStickinessCookieName)

	return method || sticky || stickiness || cookieName
}

func hasMaxConnLabels(instances []ecsInstance) bool {
	mca := hasFirst(instances, label.TraefikBackendMaxConnAmount)
	mcef := hasFirst(instances, label.TraefikBackendMaxConnExtractorFunc)
	return mca && mcef
}

// Label functions

func getFuncStringValue(labelName string, defaultValue string) func(i ecsInstance) string {
	return func(i ecsInstance) string {
		return getStringValue(i, labelName, defaultValue)
	}
}

func getFuncBoolValue(labelName string, defaultValue bool) func(i ecsInstance) bool {
	return func(i ecsInstance) bool {
		return getBoolValue(i, labelName, defaultValue)
	}
}

func getFuncSliceString(labelName string) func(i ecsInstance) []string {
	return func(i ecsInstance) []string {
		return getSliceString(i, labelName)
	}
}

func hasFuncFirst(labelName string) func(instances []ecsInstance) bool {
	return func(instances []ecsInstance) bool {
		return hasFirst(instances, labelName)
	}
}

func getFuncFirstStringValue(labelName string, defaultValue string) func(instances []ecsInstance) string {
	return func(instances []ecsInstance) string {
		return getFirstStringValue(instances, labelName, defaultValue)
	}
}

func getFuncFirstIntValue(labelName string, defaultValue int) func(instances []ecsInstance) int {
	return func(instances []ecsInstance) int {
		if len(instances) < 0 {
			return defaultValue
		}
		return getIntValue(instances[0], labelName, defaultValue)
	}
}

func getFuncFirstInt64Value(labelName string, defaultValue int64) func(instances []ecsInstance) int64 {
	return func(instances []ecsInstance) int64 {
		if len(instances) < 0 {
			return defaultValue
		}
		return getInt64Value(instances[0], labelName, defaultValue)
	}
}

func getFuncFirstBoolValue(labelName string, defaultValue bool) func(instances []ecsInstance) bool {
	return func(instances []ecsInstance) bool {
		if len(instances) < 0 {
			return defaultValue
		}
		return getBoolValue(instances[0], labelName, defaultValue)
	}
}

func getStringValue(i ecsInstance, labelName string, defaultValue string) string {
	if v, ok := i.containerDefinition.DockerLabels[labelName]; ok {
		if v == nil {
			return defaultValue
		}
		if len(*v) == 0 {
			return defaultValue
		}
		return *v
	}
	return defaultValue
}

func getBoolValue(i ecsInstance, labelName string, defaultValue bool) bool {
	rawValue, ok := i.containerDefinition.DockerLabels[labelName]
	if ok {
		if rawValue != nil {
			v, err := strconv.ParseBool(*rawValue)
			if err == nil {
				return v
			}
		}
	}
	return defaultValue
}

func getIntValue(i ecsInstance, labelName string, defaultValue int) int {
	rawValue, ok := i.containerDefinition.DockerLabels[labelName]
	if ok {
		if rawValue != nil {
			v, err := strconv.Atoi(*rawValue)
			if err == nil {
				return v
			}
		}
	}
	return defaultValue
}

func getInt64Value(i ecsInstance, labelName string, defaultValue int64) int64 {
	rawValue, ok := i.containerDefinition.DockerLabels[labelName]
	if ok {
		if rawValue != nil {
			v, err := strconv.ParseInt(*rawValue, 10, 64)
			if err == nil {
				return v
			}
		}
	}
	return defaultValue
}

func getSliceString(i ecsInstance, labelName string) []string {
	if value, ok := i.containerDefinition.DockerLabels[labelName]; ok {
		if value == nil {
			return nil
		}
		if len(*value) == 0 {
			return nil
		}
		return label.SplitAndTrimString(*value, ",")
	}
	return nil
}

func hasFirst(instances []ecsInstance, labelName string) bool {
	if len(instances) > 0 {
		v, ok := instances[0].containerDefinition.DockerLabels[labelName]
		return ok && v != nil && len(*v) != 0
	}
	return false
}

func getFirstStringValue(instances []ecsInstance, labelName string, defaultValue string) string {
	if len(instances) == 0 {
		return defaultValue
	}
	return getStringValue(instances[0], labelName, defaultValue)
}

func isEnabled(i ecsInstance, exposedByDefault bool) bool {
	return getBoolValue(i, label.TraefikEnable, exposedByDefault)
}
