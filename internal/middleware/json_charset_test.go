package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/charmap"
)

func TestJSONCharsetAllowsUTF8Cyrillic(t *testing.T) {
	router := testJSONCharsetRouter()
	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"legal_address":"Екатеринбург, Ленина 1"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() != `{"legal_address":"Екатеринбург, Ленина 1"}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestJSONCharsetConvertsWindows1251(t *testing.T) {
	encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(`{"legal_address":"Екатеринбург, Ленина 1"}`))
	if err != nil {
		t.Fatalf("encode windows-1251: %v", err)
	}
	router := testJSONCharsetRouter()
	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json; charset=windows-1251")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() != `{"legal_address":"Екатеринбург, Ленина 1"}` {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestJSONCharsetRejectsInvalidUTF8(t *testing.T) {
	router := testJSONCharsetRouter()
	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader([]byte{0xff, 0xfe}))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func testJSONCharsetRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JSONCharset())
	router.PATCH("/", func(context *gin.Context) {
		body, _ := context.GetRawData()
		context.Data(http.StatusOK, "application/json; charset=utf-8", body)
	})
	return router
}
