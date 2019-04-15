package scim

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() (*Server, error) {
	config, err := NewServiceProviderConfigFromFile("testdata/simple_service_provider_config.json")
	if err != nil {
		return nil, err
	}
	userSchema, err := NewSchemaFromFile("testdata/simple_user_schema.json")
	if err != nil {
		return nil, err
	}
	userResourceType, err := NewResourceTypeFromFile("testdata/simple_user_resource_type.json", newTestResourceHandler())
	if err != nil {
		return nil, err
	}
	server, err := NewServer(config, []Schema{userSchema}, []ResourceType{userResourceType})
	if err != nil {
		return nil, err
	}

	return &server, err
}

func TestErr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Invalid", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestServerSchemasHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Schemas", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response listResponse
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response.TotalResults != 1 {
		t.Errorf("handler returned unexpected body: got %v want 1 total result", rr.Body.String())
	}

	schemas, ok := response.Resources.([]interface{})
	if !ok {
		t.Errorf("resources is not a list of objects")
	}

	if len(schemas) != 1 {
		t.Errorf("resources contains more than one schema")
		return
	}

	schema, ok := schemas[0].(map[string]interface{})
	if !ok {
		t.Errorf("schema is not an object")
	}

	if schema["ID"].(string) != "urn:ietf:params:scim:schemas:core:2.0:User" {
		t.Errorf("schema does not contain the correct id: %v", schema["ID"])
	}
}

func TestServerSchemaHandlerInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Schemas/urn:ietf:params:scim:schemas:core:2.0:Group", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestServerSchemaHandlerValid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Schemas/urn:ietf:params:scim:schemas:core:2.0:User", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var schema schema
	if err := json.Unmarshal(rr.Body.Bytes(), &schema); err != nil {
		t.Error(err)
	}

	if schema.ID != "urn:ietf:params:scim:schemas:core:2.0:User" {
		t.Errorf("schema does not contain the correct id: %s", schema.ID)
	}
}

func TestServerResourceTypesHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ResourceTypes", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response listResponse
	json.Unmarshal(rr.Body.Bytes(), &response)
	if response.TotalResults != 1 {
		t.Errorf("handler returned unexpected body: got %v want 1 total result", rr.Body.String())
	}

	schemas, ok := response.Resources.([]interface{})
	if !ok {
		t.Errorf("resources is not a list of objects")
	}

	if len(schemas) != 1 {
		t.Errorf("resources contains more than one schema")
		return
	}

	resourceType, ok := schemas[0].(map[string]interface{})
	if !ok {
		t.Errorf("schema is not an object")
	}

	if resourceType["name"].(string) != "User" {
		t.Errorf("schema does not contain the correct id: %v", resourceType["Name"])
	}
}

func TestServerResourceTypeHandlerInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ResourceTypes/Group", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestServerResourceTypeHandlerValid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ResourceTypes/User", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resourceType resourceType
	json.Unmarshal(rr.Body.Bytes(), &resourceType)
	if resourceType.ID != "User" {
		t.Errorf("schema does not contain the correct name: %s", resourceType.Name)
	}
}

func TestServerServiceProviderConfigHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ServiceProviderConfig", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourcePostHandlerInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/Users", strings.NewReader(`{"id": "other"}`))
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourcePostHandlerValid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/Users", strings.NewReader(`{"id": "other", "userName": "test"}`))
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	var resource Resource
	json.Unmarshal(rr.Body.Bytes(), &resource)
	if resource.CoreAttributes["userName"] != "test" {
		t.Errorf("handler did not return the resource correctly")
	}
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourceGetHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Users/0001", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	var resource Resource
	json.Unmarshal(rr.Body.Bytes(), &resource)
	if resource.CoreAttributes["userName"] != "test" {
		t.Errorf("handler did not return the resource correctly")
	}
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourcesGetHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/Users", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	var response listResponse
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response.TotalResults != 1 {
		t.Errorf("handler returned unexpected body: got %v want 1 total result", rr.Body.String())
	}

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourcePutHandlerInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/Users/0001", strings.NewReader(`{"more": "test"}`))
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourcePutHandlerValid(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/Users/0001", strings.NewReader(`{"id": "test", "userName": "other"}`))
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	var resource Resource
	json.Unmarshal(rr.Body.Bytes(), &resource)
	if resource.CoreAttributes["userName"] != "other" {
		t.Errorf("handler did not replace previous resource")
	}
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestServerResourceDeleteHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/Users/0001", nil)
	rr := httptest.NewRecorder()

	server, err := newTestServer()
	if err != nil {
		t.Error(err)
	}
	server.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}
