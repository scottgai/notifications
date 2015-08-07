package campaigntypes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cloudfoundry-incubator/notifications/models"
	"github.com/cloudfoundry-incubator/notifications/v2/collections"
	"github.com/ryanmoran/stack"
)

type collectionGetter interface {
	Get(conn models.ConnectionInterface, senderID, campaignTypeID, clientID string) (collections.CampaignType, error)
}

type ShowHandler struct {
	collection collectionGetter
}

func NewShowHandler(collection collectionGetter) ShowHandler {
	return ShowHandler{
		collection: collection,
	}
}

func (h ShowHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request, context stack.Context) {
	splitURL := strings.Split(request.URL.Path, "/")
	campaignTypeID := splitURL[len(splitURL)-1]
	senderID := splitURL[len(splitURL)-3]

	if senderID == "" {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(writer, `{"errors": [%q]}`, "missing sender id")
		return
	}

	if campaignTypeID == "" {
		headers := writer.Header()
		headers.Set("Location", fmt.Sprintf("/senders/%s/campaign_types", senderID))
		writer.WriteHeader(http.StatusMovedPermanently)
		return
	}

	clientID := context.Get("client_id")
	if clientID == "" {
		writer.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(writer, `{"errors": [%q]}`, "missing client id")
		return
	}

	database := context.Get("database").(models.DatabaseInterface)
	campaignType, err := h.collection.Get(database.Connection(), campaignTypeID, senderID, context.Get("client_id").(string))
	if err != nil {
		var errorMessage string
		switch e := err.(type) {
		case collections.NotFoundError:
			errorMessage = e.Message
			writer.WriteHeader(http.StatusNotFound)
		default:
			writer.WriteHeader(http.StatusInternalServerError)
			errorMessage = err.Error()
		}

		fmt.Fprintf(writer, `{"errors": [%q]}`, errorMessage)
		return
	}

	jsonMap := map[string]interface{}{
		"id":          campaignType.ID,
		"name":        campaignType.Name,
		"description": campaignType.Description,
		"critical":    campaignType.Critical,
		"template_id": "",
	}

	jsonBody, err := json.Marshal(jsonMap)
	if err != nil {
		panic(err)
	}

	writer.Write([]byte(jsonBody))
}
