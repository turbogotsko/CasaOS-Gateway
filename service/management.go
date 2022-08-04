package service

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Gateway/common"
)

const RoutesFile = "routes.json"

type Management struct {
	pathTargetMap       map[string]string
	pathReverseProxyMap map[string]*httputil.ReverseProxy
	state               *State
}

func NewManagementService(state *State) *Management {
	routesFilepath := filepath.Join(state.GetRuntimePath(), RoutesFile)

	// try to load routes from routes.json
	pathTargetMap, err := loadPathTargetMapFrom(routesFilepath)
	if err != nil {
		log.Println(err)
		pathTargetMap = make(map[string]string)
	}

	pathReverseProxyMap := make(map[string]*httputil.ReverseProxy)

	for path, target := range pathTargetMap {
		targetURL, err := url.Parse(target)
		if err != nil {
			log.Println(err)
			continue
		}
		pathReverseProxyMap[path] = httputil.NewSingleHostReverseProxy(targetURL)
	}

	return &Management{
		pathTargetMap:       pathTargetMap,
		pathReverseProxyMap: pathReverseProxyMap,
		state:               state,
	}
}

func (g *Management) CreateRoute(route *common.Route) error {
	url, err := url.Parse(route.Target)
	if err != nil {
		return err
	}

	g.pathTargetMap[route.Path] = route.Target
	g.pathReverseProxyMap[route.Path] = httputil.NewSingleHostReverseProxy(url)

	routesFilePath := filepath.Join(g.state.GetRuntimePath(), RoutesFile)

	err = savePathTargetMapTo(routesFilePath, g.pathTargetMap)
	if err != nil {
		return err
	}

	return nil
}

func (g *Management) GetRoutes() []*common.Route {
	routes := make([]*common.Route, 0)

	for path, target := range g.pathTargetMap {
		routes = append(routes, &common.Route{
			Path:   path,
			Target: target,
		})
	}

	return routes
}

func (g *Management) GetProxy(path string) *httputil.ReverseProxy {
	// sort paths by length in descending order
	// (without this step, a path like "/abcd" can potentially be matched with "/ab")
	paths := getSortedKeys(g.pathReverseProxyMap)

	for _, p := range paths {
		if strings.HasPrefix(path, p) {
			return g.pathReverseProxyMap[p]
		}
	}
	return nil
}

func (g *Management) GetGatewayPort() string {
	return g.state.GetGatewayPort()
}

func (g *Management) SetGatewayPort(port string) error {
	if err := g.state.SetGatewayPort(port); err != nil {
		return err
	}

	return nil
}

func getSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })

	return keys
}

func loadPathTargetMapFrom(routesFilepath string) (map[string]string, error) {
	content, err := ioutil.ReadFile(routesFilepath)
	if err != nil {
		return nil, err
	}

	pathTargetMap := make(map[string]string)
	err = json.Unmarshal(content, &pathTargetMap)
	if err != nil {
		return nil, err
	}

	return pathTargetMap, nil
}

func savePathTargetMapTo(routesFilepath string, pathTargetMap map[string]string) error {
	content, err := json.Marshal(pathTargetMap)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(routesFilepath, content, 0o600)
}
