package app

import "github.com/gin-gonic/gin"

type Route struct {
	Name    string
	Method  string
	Pattern string
	APIFunc gin.HandlerFunc
}

type Routes []Route

func applyRoutes(group *gin.RouterGroup, routes []Route) {
	for _, route := range routes {
		switch route.Method {
		case "GET":
			group.GET(route.Pattern, route.APIFunc)
		case "POST":
			group.POST(route.Pattern, route.APIFunc)
		case "PUT":
			group.PUT(route.Pattern, route.APIFunc)
		case "PATCH":
			group.PATCH(route.Pattern, route.APIFunc)
		case "DELETE":
			group.DELETE(route.Pattern, route.APIFunc)
		}
	}
}

func (s *Server) registerRoutes() {
	apiGroup := s.router.Group("/api/v1")
	applyRoutes(apiGroup, s.getRoutes())
}

func (s *Server) getRoutes() Routes {
	routes := Routes{}
	routes = append(routes, s.getHealthRoutes()...)
	routes = append(routes, s.getAuthRoutes()...)
	routes = append(routes, s.getTemplateRoutes()...)
	routes = append(routes, s.getProjectRoutes()...)
	routes = append(routes, s.getConfigRoutes()...)
	routes = append(routes, s.getValidationRoutes()...)
	routes = append(routes, s.getReviewRoutes()...)
	return routes
}
