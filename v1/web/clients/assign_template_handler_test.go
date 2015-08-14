package clients_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/notifications/testing/fakes"
	"github.com/cloudfoundry-incubator/notifications/v1/web/clients"
	"github.com/cloudfoundry-incubator/notifications/v1/web/webutil"
	"github.com/ryanmoran/stack"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AssignTemplateHandler", func() {
	var (
		handler          clients.AssignTemplateHandler
		templateAssigner *fakes.TemplateAssigner
		errorWriter      *fakes.ErrorWriter
		context          stack.Context
		database         *fakes.Database
	)

	BeforeEach(func() {
		templateAssigner = fakes.NewTemplateAssigner()
		errorWriter = fakes.NewErrorWriter()
		database = fakes.NewDatabase()
		context = stack.NewContext()
		context.Set("database", database)

		handler = clients.NewAssignTemplateHandler(templateAssigner, errorWriter)
	})

	It("associates a template with a client", func() {
		body, err := json.Marshal(map[string]string{
			"template": "my-template",
		})
		if err != nil {
			panic(err)
		}

		w := httptest.NewRecorder()
		request, err := http.NewRequest("PUT", "/clients/my-client/template", bytes.NewBuffer(body))
		if err != nil {
			panic(err)
		}

		handler.ServeHTTP(w, request, context)

		Expect(w.Code).To(Equal(http.StatusNoContent))

		Expect(templateAssigner.AssignToClientCall.Receives.Database).To(Equal(database))
		Expect(templateAssigner.AssignToClientCall.Receives.ClientID).To(Equal("my-client"))
		Expect(templateAssigner.AssignToClientCall.Receives.TemplateID).To(Equal("my-template"))
	})

	It("delegates to the error writer when the assigner errors", func() {
		templateAssigner.AssignToClientCall.Returns.Error = errors.New("banana")
		body, err := json.Marshal(map[string]string{
			"template": "my-template",
		})
		if err != nil {
			panic(err)
		}

		w := httptest.NewRecorder()
		request, err := http.NewRequest("PUT", "/clients/my-client/template", bytes.NewBuffer(body))
		if err != nil {
			panic(err)
		}

		handler.ServeHTTP(w, request, context)
		Expect(errorWriter.WriteCall.Receives.Error).To(Equal(errors.New("banana")))
	})

	It("writes a ParseError to the error writer when request body is invalid", func() {
		body := []byte(`{ "this is" : not-valid-json }`)

		w := httptest.NewRecorder()
		request, err := http.NewRequest("PUT", "/clients/my-client/template", bytes.NewBuffer(body))
		if err != nil {
			panic(err)
		}

		handler.ServeHTTP(w, request, context)
		Expect(errorWriter.WriteCall.Receives.Error).To(BeAssignableToTypeOf(webutil.ParseError{}))
	})
})
