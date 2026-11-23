package httpserver

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/scim"
)

const scimBasePath = "/scim/v2"

type scimName struct {
	Formatted string `json:"formatted,omitempty"`
}

type scimEmail struct {
	Value   string `json:"value"`
	Primary bool   `json:"primary,omitempty"`
}

type scimMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
}

type scimUserResource struct {
	Schemas    []string    `json:"schemas"`
	ID         string      `json:"id"`
	ExternalID string      `json:"externalId,omitempty"`
	UserName   string      `json:"userName"`
	Name       *scimName   `json:"name,omitempty"`
	Emails     []scimEmail `json:"emails,omitempty"`
	Active     *bool       `json:"active,omitempty"`
	Meta       scimMeta    `json:"meta"`
}

type scimGroupResource struct {
	Schemas     []string     `json:"schemas"`
	ID          string       `json:"id"`
	ExternalID  string       `json:"externalId,omitempty"`
	DisplayName string       `json:"displayName"`
	Members     []scimMember `json:"members,omitempty"`
	Meta        scimMeta     `json:"meta"`
}

type scimMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
}

type scimListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    any      `json:"Resources"`
}

type scimPatch struct {
	Schemas    []string `json:"schemas"`
	Operations []struct {
		Op    string          `json:"op"`
		Path  string          `json:"path"`
		Value json.RawMessage `json:"value"`
	} `json:"Operations"`
}

func (s *Server) registerSCIMRoutes(engine *gin.Engine) {
	if engine == nil || s.scimStore == nil || s.scimToken == "" {
		return
	}
	engine.GET(scimBasePath+"/ServiceProviderConfig", s.scimAuth, s.scimServiceProviderConfig)
	engine.GET(scimBasePath+"/ResourceTypes", s.scimAuth, s.scimResourceTypes)
	engine.GET(scimBasePath+"/Users", s.scimAuth, s.scimListUsers)
	engine.POST(scimBasePath+"/Users", s.scimAuth, s.scimCreateUser)
	engine.GET(scimBasePath+"/Users/:id", s.scimAuth, s.scimGetUser)
	engine.PUT(scimBasePath+"/Users/:id", s.scimAuth, s.scimReplaceUser)
	engine.PATCH(scimBasePath+"/Users/:id", s.scimAuth, s.scimPatchUser)
	engine.DELETE(scimBasePath+"/Users/:id", s.scimAuth, s.scimDeleteUser)
	engine.GET(scimBasePath+"/Groups", s.scimAuth, s.scimListGroups)
	engine.POST(scimBasePath+"/Groups", s.scimAuth, s.scimCreateGroup)
	engine.GET(scimBasePath+"/Groups/:id", s.scimAuth, s.scimGetGroup)
	engine.PUT(scimBasePath+"/Groups/:id", s.scimAuth, s.scimReplaceGroup)
	engine.PATCH(scimBasePath+"/Groups/:id", s.scimAuth, s.scimPatchGroup)
	engine.DELETE(scimBasePath+"/Groups/:id", s.scimAuth, s.scimDeleteGroup)
}

func (s *Server) scimAuth(context *gin.Context) {
	if s.scimToken == "" {
		context.Status(http.StatusNotFound)
		context.Abort()
		return
	}
	authorization := strings.TrimSpace(context.GetHeader("Authorization"))
	token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if authorization == "" || !strings.HasPrefix(authorization, "Bearer ") || subtle.ConstantTimeCompare([]byte(token), []byte(s.scimToken)) != 1 {
		writeSCIMError(context, http.StatusUnauthorized, "invalid SCIM bearer token")
		context.Abort()
		return
	}
	context.Next()
}

func (s *Server) scimServiceProviderConfig(context *gin.Context) {
	writeSCIMJSON(context, http.StatusOK, gin.H{
		"schemas":        []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":          gin.H{"supported": true},
		"bulk":           gin.H{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":         gin.H{"supported": true, "maxResults": 200},
		"changePassword": gin.H{"supported": false},
		"sort":           gin.H{"supported": false},
		"etag":           gin.H{"supported": false},
		"authenticationSchemes": []gin.H{{
			"type":        "oauthbearertoken",
			"name":        "Bearer",
			"description": "Skill Dock SCIM bearer token",
		}},
	})
}

func (s *Server) scimResourceTypes(context *gin.Context) {
	writeSCIMJSON(context, http.StatusOK, gin.H{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"Resources": []gin.H{
			{"id": "User", "name": "User", "endpoint": "/Users", "schema": "urn:ietf:params:scim:schemas:core:2.0:User"},
			{"id": "Group", "name": "Group", "endpoint": "/Groups", "schema": "urn:ietf:params:scim:schemas:core:2.0:Group"},
		},
	})
}

func (s *Server) scimListUsers(context *gin.Context) {
	start, count := scimPage(context)
	users, total, err := s.scimStore.ListUsers(context.Request.Context(), context.Query("filter"), start, count)
	if err != nil {
		writeSCIMError(context, http.StatusInternalServerError, "could not list users")
		return
	}
	resources := make([]scimUserResource, 0, len(users))
	for _, user := range users {
		resources = append(resources, toSCIMUser(user))
	}
	writeSCIMJSON(context, http.StatusOK, scimListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: total,
		StartIndex:   start,
		ItemsPerPage: len(resources),
		Resources:    resources,
	})
}

func (s *Server) scimCreateUser(context *gin.Context) {
	var payload scimUserResource
	if err := context.ShouldBindJSON(&payload); err != nil {
		writeSCIMError(context, http.StatusBadRequest, "invalid SCIM user")
		return
	}
	user, err := s.scimStore.CreateUser(context.Request.Context(), fromSCIMUser(payload))
	if err != nil {
		writeSCIMError(context, http.StatusConflict, err.Error())
		return
	}
	writeSCIMJSON(context, http.StatusCreated, toSCIMUser(user))
}

func (s *Server) scimGetUser(context *gin.Context) {
	user, err := s.scimStore.GetUser(context.Request.Context(), context.Param("id"))
	if err != nil {
		writeSCIMError(context, http.StatusNotFound, "user not found")
		return
	}
	writeSCIMJSON(context, http.StatusOK, toSCIMUser(user))
}

func (s *Server) scimReplaceUser(context *gin.Context) {
	var payload scimUserResource
	if err := context.ShouldBindJSON(&payload); err != nil {
		writeSCIMError(context, http.StatusBadRequest, "invalid SCIM user")
		return
	}
	user := fromSCIMUser(payload)
	user.ID = context.Param("id")
	updated, err := s.scimStore.ReplaceUser(context.Request.Context(), user)
	if err != nil {
		writeSCIMError(context, http.StatusNotFound, "user not found")
		return
	}
	writeSCIMJSON(context, http.StatusOK, toSCIMUser(updated))
}

func (s *Server) scimPatchUser(context *gin.Context) {
	user, err := s.scimStore.GetUser(context.Request.Context(), context.Param("id"))
	if err != nil {
		writeSCIMError(context, http.StatusNotFound, "user not found")
		return
	}
	var patch scimPatch
	if err := context.ShouldBindJSON(&patch); err != nil {
		writeSCIMError(context, http.StatusBadRequest, "invalid SCIM patch")
		return
	}
	for _, operation := range patch.Operations {
		path := strings.ToLower(strings.TrimSpace(operation.Path))
		switch {
		case path == "active" || path == "urn:ietf:params:scim:schemas:core:2.0:user:active":
			var active bool
			if json.Unmarshal(operation.Value, &active) == nil {
				user.Active = active
			}
		case path == "displayname" || path == "name.formatted":
			var value string
			if json.Unmarshal(operation.Value, &value) == nil {
				user.DisplayName = value
			}
		case path == "username":
			var value string
			if json.Unmarshal(operation.Value, &value) == nil {
				user.UserName = value
			}
		}
	}
	updated, err := s.scimStore.ReplaceUser(context.Request.Context(), user)
	if err != nil {
		writeSCIMError(context, http.StatusInternalServerError, "could not update user")
		return
	}
	writeSCIMJSON(context, http.StatusOK, toSCIMUser(updated))
}

func (s *Server) scimDeleteUser(context *gin.Context) {
	if err := s.scimStore.DeleteUser(context.Request.Context(), context.Param("id")); err != nil {
		writeSCIMError(context, http.StatusInternalServerError, "could not delete user")
		return
	}
	context.Status(http.StatusNoContent)
}

func (s *Server) scimListGroups(context *gin.Context) {
	start, count := scimPage(context)
	groups, total, err := s.scimStore.ListGroups(context.Request.Context(), start, count)
	if err != nil {
		writeSCIMError(context, http.StatusInternalServerError, "could not list groups")
		return
	}
	resources := make([]scimGroupResource, 0, len(groups))
	for _, group := range groups {
		resources = append(resources, toSCIMGroup(group))
	}
	writeSCIMJSON(context, http.StatusOK, scimListResponse{
		Schemas:      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: total,
		StartIndex:   start,
		ItemsPerPage: len(resources),
		Resources:    resources,
	})
}

func (s *Server) scimCreateGroup(context *gin.Context) {
	var payload scimGroupResource
	if err := context.ShouldBindJSON(&payload); err != nil {
		writeSCIMError(context, http.StatusBadRequest, "invalid SCIM group")
		return
	}
	group, err := s.scimStore.CreateGroup(context.Request.Context(), fromSCIMGroup(payload))
	if err != nil {
		writeSCIMError(context, http.StatusConflict, err.Error())
		return
	}
	writeSCIMJSON(context, http.StatusCreated, toSCIMGroup(group))
}

func (s *Server) scimGetGroup(context *gin.Context) {
	group, err := s.scimStore.GetGroup(context.Request.Context(), context.Param("id"))
	if err != nil {
		writeSCIMError(context, http.StatusNotFound, "group not found")
		return
	}
	writeSCIMJSON(context, http.StatusOK, toSCIMGroup(group))
}

func (s *Server) scimReplaceGroup(context *gin.Context) {
	var payload scimGroupResource
	if err := context.ShouldBindJSON(&payload); err != nil {
		writeSCIMError(context, http.StatusBadRequest, "invalid SCIM group")
		return
	}
	group := fromSCIMGroup(payload)
	group.ID = context.Param("id")
	updated, err := s.scimStore.ReplaceGroup(context.Request.Context(), group)
	if err != nil {
		writeSCIMError(context, http.StatusNotFound, "group not found")
		return
	}
	writeSCIMJSON(context, http.StatusOK, toSCIMGroup(updated))
}

func (s *Server) scimPatchGroup(context *gin.Context) {
	group, err := s.scimStore.GetGroup(context.Request.Context(), context.Param("id"))
	if err != nil {
		writeSCIMError(context, http.StatusNotFound, "group not found")
		return
	}
	var patch scimPatch
	if err := context.ShouldBindJSON(&patch); err != nil {
		writeSCIMError(context, http.StatusBadRequest, "invalid SCIM patch")
		return
	}
	for _, operation := range patch.Operations {
		path := strings.ToLower(strings.TrimSpace(operation.Path))
		if path == "displayname" {
			var value string
			if json.Unmarshal(operation.Value, &value) == nil {
				group.DisplayName = value
			}
			continue
		}
		if path != "" && path != "members" {
			continue
		}
		var members []scimMember
		if json.Unmarshal(operation.Value, &members) != nil {
			continue
		}
		switch strings.ToLower(operation.Op) {
		case "add", "replace":
			for _, member := range members {
				group.Members = appendMember(group.Members, scim.Member{Value: member.Value, Display: member.Display})
			}
		case "remove":
			for _, member := range members {
				group.Members = removeMember(group.Members, member.Value)
			}
		}
	}
	updated, err := s.scimStore.ReplaceGroup(context.Request.Context(), group)
	if err != nil {
		writeSCIMError(context, http.StatusInternalServerError, "could not update group")
		return
	}
	writeSCIMJSON(context, http.StatusOK, toSCIMGroup(updated))
}

func (s *Server) scimDeleteGroup(context *gin.Context) {
	if err := s.scimStore.DeleteGroup(context.Request.Context(), context.Param("id")); err != nil {
		writeSCIMError(context, http.StatusInternalServerError, "could not delete group")
		return
	}
	context.Status(http.StatusNoContent)
}

func scimPage(context *gin.Context) (int, int) {
	start := 1
	count := 100
	if parsed, err := strconv.Atoi(context.Query("startIndex")); err == nil && parsed > 0 {
		start = parsed
	}
	if parsed, err := strconv.Atoi(context.Query("count")); err == nil && parsed > 0 {
		count = parsed
	}
	return start, count
}

func toSCIMUser(user scim.User) scimUserResource {
	active := user.Active
	resource := scimUserResource{
		Schemas:    []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		ID:         user.ID,
		ExternalID: user.ExternalID,
		UserName:   user.UserName,
		Active:     &active,
		Meta: scimMeta{
			ResourceType: "User",
			Created:      user.CreatedAt.UTC().Format(time.RFC3339),
			LastModified: user.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}
	if user.DisplayName != "" {
		resource.Name = &scimName{Formatted: user.DisplayName}
	}
	if user.Email != "" {
		resource.Emails = []scimEmail{{Value: user.Email, Primary: true}}
	}
	return resource
}

func fromSCIMUser(user scimUserResource) scim.User {
	result := scim.User{
		ID:         user.ID,
		ExternalID: user.ExternalID,
		UserName:   user.UserName,
		Active:     true,
	}
	if user.Active != nil {
		result.Active = *user.Active
	}
	if user.Name != nil {
		result.DisplayName = user.Name.Formatted
	}
	if len(user.Emails) > 0 {
		result.Email = user.Emails[0].Value
	}
	return result
}

func toSCIMGroup(group scim.Group) scimGroupResource {
	members := make([]scimMember, 0, len(group.Members))
	for _, member := range group.Members {
		members = append(members, scimMember{Value: member.Value, Display: member.Display})
	}
	return scimGroupResource{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		ID:          group.ID,
		ExternalID:  group.ExternalID,
		DisplayName: group.DisplayName,
		Members:     members,
		Meta: scimMeta{
			ResourceType: "Group",
			Created:      group.CreatedAt.UTC().Format(time.RFC3339),
			LastModified: group.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}
}

func fromSCIMGroup(group scimGroupResource) scim.Group {
	members := make([]scim.Member, 0, len(group.Members))
	for _, member := range group.Members {
		members = append(members, scim.Member{Value: member.Value, Display: member.Display})
	}
	return scim.Group{
		ID:          group.ID,
		ExternalID:  group.ExternalID,
		DisplayName: group.DisplayName,
		Members:     members,
	}
}

func appendMember(members []scim.Member, member scim.Member) []scim.Member {
	for _, current := range members {
		if current.Value == member.Value {
			return members
		}
	}
	return append(members, member)
}

func removeMember(members []scim.Member, value string) []scim.Member {
	result := members[:0]
	for _, current := range members {
		if current.Value != value {
			result = append(result, current)
		}
	}
	return result
}

func writeSCIMJSON(context *gin.Context, status int, payload any) {
	context.Header("Content-Type", "application/scim+json")
	context.Status(status)
	_ = json.NewEncoder(context.Writer).Encode(payload)
}

func writeSCIMError(context *gin.Context, status int, detail string) {
	writeSCIMJSON(context, status, gin.H{
		"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		"status":  strconv.Itoa(status),
		"detail":  detail,
	})
}
