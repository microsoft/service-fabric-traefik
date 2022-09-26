package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/ghodss/yaml"
	sf "github.com/jjcollinge/servicefabric"
	"github.com/microsoft/service-fabric-traefik/serviceFabricDiscoveryService/pkg/certstorehelper"
	log "github.com/sirupsen/logrus"
	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

const (
	traefikServiceFabricExtensionKey     = "Traefik"
	traefikSFEnableLabelOverrides        = "Traefik.enableLabelOverrides"
	traefikSFEnableLabelOverridesDefault = true

	kindStateful  = "Stateful"
	kindStateless = "Stateless"
)

// Config the plugin configuration.
type Config struct {
	PollInterval   string `json:"pollInterval,omitempty"`
	HttpEntrypoint string `json:"httpEntrypoint,omitempty"`

	ClusterManagementURL string `json:"clusterManagementURL,omitempty"`
	Certificate          string `json:"certificate,omitempty"`
	CertificateKey       string `json:"certificateKey,omitempty"`
	CertStoreSearchKey   string `json:"certStoreSearchKey,omitempty"`
	InsecureSkipVerify   bool   `json:"insecureSkipVerify,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		PollInterval:   "5s",
		HttpEntrypoint: "web",
	}
}

// Provider a simple provider plugin.
type Provider struct {
	name           string
	pollInterval   time.Duration
	httpEntrypoint string
	tcpEntrypoint  string

	clusterManagementURL string
	apiVersion           string
	tlsConfig            *certstorehelper.ClientTLS
	sfClient             sfClient

	cancel func()
}

// New creates a new Provider plugin.
func NewDiscoveryWorker(ctx context.Context, config *Config, name string) (*Provider, error) {
	pi, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		name:                 name,
		apiVersion:           sf.DefaultAPIVersion,
		pollInterval:         pi,
		clusterManagementURL: config.ClusterManagementURL,
		httpEntrypoint:       config.HttpEntrypoint,
	}

	if strings.HasPrefix(p.clusterManagementURL, "https") &&
		(config.CertStoreSearchKey != "" || (config.CertificateKey != "" && config.Certificate != "")) {
		p.tlsConfig = &certstorehelper.ClientTLS{
			Cert:               config.Certificate,
			Key:                config.CertificateKey,
			CertStoreSearchKey: config.CertStoreSearchKey,
			InsecureSkipVerify: config.InsecureSkipVerify,
		}
	}

	return p, nil
}

// Init the provider.
func (p *Provider) Init() error {
	var err error
	if p.pollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}

	log.Printf("Initializing: %s, version: %s", p.clusterManagementURL, p.apiVersion)

	tlsConfig, err := p.tlsConfig.CreateTLSConfig()
	if err != nil {
		return err
	}

	sfClient, err := sf.NewClient(&http.Client{Timeout: 5 * time.Second}, p.clusterManagementURL, p.apiVersion, tlsConfig)
	if err != nil {
		return err
	}
	p.sfClient = sfClient

	return nil
}

// Provide creates and send dynamic configuration.
func (p *Provider) Provide(cfgChan chan<- []byte) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		p.loadConfiguration(ctx, cfgChan)
	}()

	return nil
}

// Stop to stop the provider and the related go routines.
func (p *Provider) Stop() error {
	p.cancel()
	return nil
}

func (p *Provider) loadConfiguration(ctx context.Context, cfgChan chan<- []byte) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e, err := p.fetchState()
			if err != nil {
				log.Print(err)
				continue
			}

			conf := p.generateConfiguration(e)

			y, err := yaml.Marshal(conf)
			if err != nil {
				return
			}

			cfgChan <- y

		case <-ctx.Done():
			return
		}
	}
}

// Normalize Replace all special chars with `-`.
func normalize(name string) string {
	fargs := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	// get function
	return strings.Join(strings.FieldsFunc(name, fargs), "-")
}

var iii int = 0

func (p *Provider) fetchState() ([]ServiceItemExtended, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
		}
	}()

	log.Print("Fetching state from cluster")

	apps, err := p.sfClient.GetApplications()
	if err != nil {
		log.Printf("failed to gets applications %v", err)
		return nil, nil
	}

	var results []ServiceItemExtended
	for _, app := range apps.Items {
		services, err := p.sfClient.GetServices(app.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get services: %w", err)
		}

		for _, service := range services.Items {
			item := ServiceItemExtended{
				ServiceItem: service,
				Application: app,
			}

			labels, err := getLabels(p.sfClient, service, app)
			if err != nil {
				log.Printf("failed to get labels: %v", err)
				continue
			}
			if len(labels) == 0 { //|| !GetBoolValue(labels, traefikSFEnableService, false) {
				continue
			}
			item.Labels = labels

			partitions, err := p.sfClient.GetPartitions(app.ID, service.ID)
			if err != nil {
				log.Printf("failed to get partitions: %v", err)
			}

			for _, partition := range partitions.Items {
				partitionExt := PartitionItemExtended{PartitionItem: partition}

				switch {
				case isStateful(item):
					partitionExt.Replicas = getValidReplicas(p.sfClient, app, service, partition)
				case isStateless(item):
					partitionExt.Instances = getValidInstances(p.sfClient, app, service, partition)
				default:
					log.Printf("Unsupported service kind %s in service %s", partition.ServiceKind, service.Name)
					continue
				}

				item.Partitions = append(item.Partitions, partitionExt)
			}

			results = append(results, item)
		}
	}

	return results, nil
}

func getValidReplicas(sfClient sfClient, app sf.ApplicationItem, service sf.ServiceItem, partition sf.PartitionItem) []sf.ReplicaItem {
	var validReplicas []sf.ReplicaItem

	if replicas, err := sfClient.GetReplicas(app.ID, service.ID, partition.PartitionInformation.ID); err != nil {
		log.Print(err)
	} else {
		for _, instance := range replicas.Items {
			if isHealthy(instance.ReplicaItemBase) && hasHTTPEndpoint(instance.ReplicaItemBase) {
				validReplicas = append(validReplicas, instance)
			}
		}
	}
	return validReplicas
}

func getValidInstances(sfClient sfClient, app sf.ApplicationItem, service sf.ServiceItem, partition sf.PartitionItem) []sf.InstanceItem {
	var validInstances []sf.InstanceItem

	if instances, err := sfClient.GetInstances(app.ID, service.ID, partition.PartitionInformation.ID); err != nil {
		log.Print(err)
	} else {
		for _, instance := range instances.Items {
			if isHealthy(instance.ReplicaItemBase) && hasHTTPEndpoint(instance.ReplicaItemBase) {
				validInstances = append(validInstances, instance)
			}
		}
	}
	return validInstances
}

func isPrimary(instanceData *sf.ReplicaItemBase) bool {
	return instanceData.ReplicaRole == "Primary"
}

func isHealthy(instanceData *sf.ReplicaItemBase) bool {
	return instanceData != nil && (instanceData.ReplicaStatus == "Ready" && instanceData.HealthState != "Error")
}

func hasHTTPEndpoint(instanceData *sf.ReplicaItemBase) bool {
	_, err := getReplicaDefaultEndpoint(instanceData)
	return err == nil
}

func getReplicaDefaultEndpoint(replicaData *sf.ReplicaItemBase) (string, error) {
	endpoints, err := decodeEndpointData(replicaData.Address)
	if err != nil {
		return "", err
	}

	var defaultHTTPEndpoint string
	for _, v := range endpoints {
		if strings.Contains(v, "http") {
			defaultHTTPEndpoint = v
			break
		}
	}

	if len(defaultHTTPEndpoint) == 0 {
		return "", errors.New("no default endpoint found")
	}
	return defaultHTTPEndpoint, nil
}

func getReplicaEndpoint(epName string, replicaData *sf.ReplicaItemBase) (string, error) {
	endpoints, err := decodeEndpointData(replicaData.Address)
	if err != nil {
		return "", err
	}

	var address string
	for k, v := range endpoints {
		if k == epName {
			if strings.Contains(v, "http") {
				address = v
				break
			}
		}
	}

	if len(address) == 0 {
		return "", errors.New("no address for endpoint found")
	}
	return address, nil
}

func decodeEndpointData(endpointData string) (map[string]string, error) {
	var endpointsMap map[string]map[string]string

	if endpointData == "" {
		return nil, errors.New("endpoint data is empty")
	}

	err := json.Unmarshal([]byte(endpointData), &endpointsMap)
	if err != nil {
		return nil, err
	}

	endpoints, endpointsExist := endpointsMap["Endpoints"]
	if !endpointsExist {
		return nil, errors.New("endpoint doesn't exist in endpoint data")
	}

	return endpoints, nil
}

func isStateful(service ServiceItemExtended) bool {
	return service.ServiceKind == kindStateful
}

func isStateless(service ServiceItemExtended) bool {
	return service.ServiceKind == kindStateless
}

// Return a set of labels from the Extension and Property manager
// Allow Extension labels to disable importing labels from the property manager.
func getLabels(sfClient sfClient, service sf.ServiceItem, app sf.ApplicationItem) (map[string]string, error) {
	labels, err := sfClient.GetServiceExtensionMap(&service, &app, traefikServiceFabricExtensionKey)
	if err != nil {
		return nil, fmt.Errorf("error retrieving serviceExtensionMap: %w", err)
	}

	if GetBoolValue(labels, traefikSFEnableLabelOverrides, traefikSFEnableLabelOverridesDefault) {
		if exists, properties, err := sfClient.GetProperties(service.ID); err == nil && exists {
			for key, value := range properties {
				labels[key] = value
			}
		}
	}
	return labels, nil
}

func TestRawRun() {
	conf := dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:           make(map[string]*dynamic.Router),
			Middlewares:       make(map[string]*dynamic.Middleware),
			Services:          make(map[string]*dynamic.Service),
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  make(map[string]*dynamic.TCPRouter),
			Services: make(map[string]*dynamic.TCPService),
		},
		TLS: &dynamic.TLSConfiguration{
			Stores:  make(map[string]tls.Store),
			Options: make(map[string]tls.Options),
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  make(map[string]*dynamic.UDPRouter),
			Services: make(map[string]*dynamic.UDPService),
		},
	}

	kvs := []*KVPair{
		{Key: "traefik.http.routers.ddd.rule", Value: "Prefix('/api')"},
		{Key: "traefik.http.middlewares.sf-stripprefixregex_nonpartitioned.stripPrefixRegex.Regex.0", Value: "^/[^/]*/[^/]*/*"},
	}

	err := Decode(kvs, &conf, "traefik")
	if err != nil {
		log.Printf("failed processing kvs entries: %v", err)
	}

	jsonData, _ := json.MarshalIndent(conf, "", "\t")

	log.Printf("Done: [%s]", string(jsonData))
}

func (p *Provider) generateConfiguration(serviceItems []ServiceItemExtended) *dynamic.Configuration {

	kvs := []*KVPair{
		//{Key: "traefik.http.middlewares.111.stripPrefix.prefixes", Value: "/api , /api1"},
		//{Key: "traefik.http.middlewares.111.stripPrefix.Prefixes.1", Value: "/api"},
		//{Key: "traefik.http.routers.ddd.rule", Value: "PathPrefix(`/api9`)"},
		{Key: "traefik.http.middlewares.sf-stripprefixregex_nonpartitioned.stripPrefixRegex.Regex", Value: "^/[^/]*/[^/]*/*"},
	}

	for _, serviceItem := range serviceItems {
		/*kv := map[string]string{
			"traefik.http.ep1": "true",
			//"traefik.http.ep1.router.rule":                           "PathPrefix(`/api`)",
			//"traefik.http.ep1.middlewares.1.stripPrefix.prefixes":    "/api",
			"traefik.http.ep1.service.loadbalancer.passhostheader":       "false",
			"traefik.http.ep1.service.loadbalancer.healthcheck.path":     "/",
			"traefik.http.ep1.service.loadbalancer.healthcheck.interval": "10s",
		}
		*/

		// get a map of endpoints names to a template for the rules
		endpoints, err := p.getKVItemsFromLabels(serviceItem.Labels)
		if err != nil {
			log.Printf("failed processing labels to kvs entries for service [%s]: %v", serviceItem.Name, err)
			continue
		}

		// generate the predefined rules for the endpoints
		for epName, ep := range endpoints {
			rules := map[string]string{}

			baseName := strings.ReplaceAll(serviceItem.Name, "/", "-")
			baseName = normalize(baseName)
			baseName = fmt.Sprintf("%s-%s", baseName, epName)

			// If there is only one partition, expose the service name route directly
			if len(serviceItem.Partitions) == 1 {
				// Expose both a default endpoint and the QP endpoint
				if ep.protocol == "http" {
					rule := fmt.Sprintf("PathPrefix(`/%s`)", serviceItem.ID)
					p.generateHTTPRuleEntries(epName, ep.rules, baseName, rule, serviceItem.Partitions[0], rules)
				} else if ep.protocol == "tcp" {
					rule := "HostSNI(`*`)"
					p.generateTCPRuleEntries(epName, ep.rules, baseName, rule, serviceItem.Partitions[0], rules)
				}
			}

			// Partition support only for stateful services using the http protocol			
			if isStateful(serviceItem) && ep.protocol == "http" {
				// Create the traefik services based on the sf service partitions
				for _, part := range serviceItem.Partitions {
					partitionID := part.PartitionInformation.ID
					name := fmt.Sprintf("%s-%s", baseName, partitionID)
					rule := fmt.Sprintf("PathPrefix(`/%s`) && Query(`PartitionID=%s`)", serviceItem.ID, partitionID)

					p.generateHTTPRuleEntries(epName, ep.rules, name, rule, part, rules)
				}
			}
			
			// pass the tls and serversTransport rules directly without proccesing
			if ep.protocol == "tls" || ep.protocol == "serversTransport" {
				for _, entry := range ep.rules {
					rules[entry.Key] = entry.Value
				}
			}

			kvsService := []*KVPair{}
			for k, v := range rules {
				kvsService = append(kvsService, &KVPair{k, v})
			}
			// validate partial/service configuration, skip them if it fails
			_, err := p.getConfigFromKeys(kvsService)
			if err != nil {
				log.Printf("failed processing kvs entries for service [%s]: %v", baseName, err)
				continue
			}

			kvs = append(kvs, kvsService...)
		}
	}

	// Finally return the right objects
	conf, err := p.getConfigFromKeys(kvs)
	if err != nil {
		log.Printf("failed processing kvs entries: %v", err)
		return nil
	}

	return conf
}

type ProtocolRules struct {
	protocol string
	rules    []*KVPair
}

// getKVItemsFromLabels generates a map of endpoints names to a template for the rules
func (p *Provider) getKVItemsFromLabels(kv map[string]string) (map[string]*ProtocolRules, error) {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	endpoints := map[string]*ProtocolRules{}
	for _, k := range keys {
		t := strings.Split(k, ".")

		if len(t) <= 2 || t[0] != "traefik" {
			continue
		}

		// TODO: Add more robust label validation
		if t[1] != "http" && t[1] != "tcp" && t[1] != "tls" {
			continue
		}

		if len(t) == 3 && kv[k] == "true" {
			if t[1] == "http" || t[1] == "tcp" {
				endpoints[t[1]+"-"+t[2]] = &ProtocolRules{t[1], []*KVPair{}}
			}
		} else {
			protoRules, ok := endpoints[t[1]+"-"+t[2]]

			if t[1] == "tls" {
				if _, ok := endpoints[t[0]+"-"+t[1]]; !ok {
					// Add global traefik tls endpoint that can be referenced by router tls options
					endpoints[t[0]+"-"+t[1]] = &ProtocolRules{t[1], []*KVPair{}}
				}
				protoRules, ok = endpoints[t[0]+"-"+t[1]]
			}

			if t[2] == "serversTransport" {
				if _, ok := endpoints[t[1]+"-"+t[2]]; !ok {
					// Add global http serversTransport endpoint that can be referenced by service loadbalancer
					endpoints[t[1]+"-"+t[2]] = &ProtocolRules{t[2], []*KVPair{}}
				}
				protoRules, ok = endpoints[t[1]+"-"+t[2]]
			}

			if !ok {
				continue
			}
			switch t[1] {
			case "http":
				if len(t) >= 5 {
					switch t[2] {
					case "serversTransport":
						// Global http connection config referenced by service loadbalancer
						// traefik.http.serversTransport.serversTransport0.serverName
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.http.serversTransports.%s", strings.Join(t[3:], ".")), Value: kv[k]})
					}
					switch t[3] {
					case "router":
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.http.routers.[SERVICE].%s", strings.Join(t[4:], ".")), Value: kv[k]})
					case "middlewares":
						//traefik.http.ep1.middlewares.1.stripPrefix.prefixes.0
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.http.middlewares.[SERVICE]-%s.%s", t[4], strings.Join(t[5:], ".")), Value: kv[k]})
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.http.routers.[SERVICE].middlewares"), Value: "[SERVICE]-" + t[4]})
					case "service":
						//traefik.http.ep1.service.loadbalancer.healthcheck.path
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.http.services.[SERVICE].%s", strings.Join(t[4:], ".")), Value: kv[k]})
					default:
					}
				}
			case "tcp":
				if len(t) >= 5 {
					switch t[3] {
					case "router":
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.tcp.routers.[SERVICE].%s", strings.Join(t[4:], ".")), Value: kv[k]})
					case "service":
						//traefik.tcp.ep1.service.loadbalancer.terminationDelay
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.tcp.services.[SERVICE].%s", strings.Join(t[4:], ".")), Value: kv[k]})
					default:
					}
				}
			case "tls":
				if len(t) >= 5 {
					//traefik.tls.option.option0.minVersion
					//traefik.tls.store.store0.defaultCertificate.certFile
					//traefik.tls.certificate.certficate0.certFile
					switch t[2] {
					case "option":
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.tls.options.%s", strings.Join(t[3:], ".")), Value: kv[k]})
					case "store":
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.tls.stores.%s", strings.Join(t[3:], ".")), Value: kv[k]})
					case "certificate":
						protoRules.rules = append(protoRules.rules, &KVPair{Key: fmt.Sprintf("traefik.tls.certificates.%s", strings.Join(t[3:], ".")), Value: kv[k]})
					default:
					}
				}
			}
		}
	}
	return endpoints, nil
}

func (p *Provider) getConfigFromKeys(kvs []*KVPair) (*dynamic.Configuration, error) {
	conf := &dynamic.Configuration{}

	err := Decode(kvs, conf, "traefik")
	if err != nil {
		log.Printf("failed processing kvs entries: %v", err)
		return nil, err
	}
	return conf, err
}

func (p *Provider) generateHTTPRuleEntries(epName string, ep []*KVPair, name string, rule string, part PartitionItemExtended, rules map[string]string) {
	// Populate the target endpoints
	addedEndpoint := false
	if part.ServiceKind == kindStateless {
		for i, instance := range part.Instances {
			url, err := getReplicaDefaultEndpoint(instance.ReplicaItemBase)
			if err == nil && url != "" {
				addedEndpoint = true
				rules[fmt.Sprintf("traefik.http.services.%s.loadbalancer.servers.%d.url", name, i)] = url
			}
		}
	} else if part.ServiceKind == kindStateful {
		for i, replica := range part.Replicas {
			if isPrimary(replica.ReplicaItemBase) && isHealthy(replica.ReplicaItemBase) {
				url, err := getReplicaDefaultEndpoint(replica.ReplicaItemBase)
				if err == nil && url != "" {
					addedEndpoint = true
					rules[fmt.Sprintf("traefik.http.services.%s.loadbalancer.servers.%d.url", name, i)] = url
				}
			}
		}
	}

	if addedEndpoint {
		rules[fmt.Sprintf("traefik.http.routers.%s.entryPoints", name)] = p.httpEntrypoint
		rules[fmt.Sprintf("traefik.http.routers.%s.service", name)] = name
		rules[fmt.Sprintf("traefik.http.routers.%s.rule", name)] = rule

		rules[fmt.Sprintf("traefik.http.routers.%s.middlewares", name)] = "sf-stripprefixregex_nonpartitioned"

		// add the user provided ones
		for _, entry := range ep {
			k := strings.Replace(entry.Key, "[SERVICE]", name, 1)
			v := strings.Replace(entry.Value, "[SERVICE]", name, 1)
			rules[k] = v
		}
	}
}

func (p *Provider) generateTCPRuleEntries(epName string, ep []*KVPair, name string, rule string, part PartitionItemExtended, rules map[string]string) {
	// Populate the target endpoints
	addedEndpoint := false
	if part.ServiceKind == kindStateless {
		for i, instance := range part.Instances {
			url, err := getReplicaDefaultEndpoint(instance.ReplicaItemBase)
			if err != nil {
				continue
			}
			host, _, err := parseRawURL(url)
			if err == nil && url != "" {
				addedEndpoint = true
				rules[fmt.Sprintf("traefik.tcp.services.%s.loadbalancer.servers.%d.address", name, i)] = host
			}
		}
	} else if part.ServiceKind == kindStateful {
		for i, replica := range part.Replicas {
			if isPrimary(replica.ReplicaItemBase) && isHealthy(replica.ReplicaItemBase) {
				url, err := getReplicaDefaultEndpoint(replica.ReplicaItemBase)
				if err != nil {
					continue
				}
				host, _, err := parseRawURL(url)
				if err == nil && url != "" {
					addedEndpoint = true
					rules[fmt.Sprintf("traefik.tcp.services.%s.loadbalancer.servers.%d.address", name, i)] = host
				}
			}
		}
	}

	if addedEndpoint {
		//rules[fmt.Sprintf("traefik.tcp.routers.%s.entryPoints", name) = "tcpport"
		rules[fmt.Sprintf("traefik.tcp.routers.%s.service", name)] = name
		rules[fmt.Sprintf("traefik.tcp.routers.%s.rule", name)] = rule

		// add the user provided ones
		for _, entry := range ep {
			entry.Key = strings.Replace(entry.Key, "[SERVICE]", name, 1)
			entry.Value = strings.Replace(entry.Value, "[SERVICE]", name, 1)
			rules[entry.Key] = entry.Value
		}
	}
}

func parseRawURL(rawurl string) (host string, scheme string, err error) {
	u, err := url.ParseRequestURI(rawurl)
	if err != nil || u.Host == "" {
		u, repErr := url.ParseRequestURI("tcp://" + rawurl)
		if repErr != nil {
			fmt.Printf("Could not parse raw url: %s, error: %v", rawurl, err)
			return "", "", err
		}
		err = nil
		return u.Host, u.Scheme, nil
	}

	return u.Host, u.Scheme, nil
}

func boolPtr(v bool) *bool {
	return &v
}
