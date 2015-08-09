package services_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/testing/fakes"
	"github.com/cloudfoundry-incubator/notifications/v1/services"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Everyone Strategy", func() {
	var (
		strategy            services.EveryoneStrategy
		tokenLoader         *fakes.TokenLoader
		token               string
		allUsers            *fakes.AllUsers
		enqueuer            *fakes.Enqueuer
		conn                *fakes.Connection
		requestReceivedTime time.Time
	)

	BeforeEach(func() {
		requestReceivedTime, _ = time.Parse(time.RFC3339Nano, "2015-06-08T14:38:03.180764129-07:00")
		conn = fakes.NewConnection()
		tokenHeader := map[string]interface{}{
			"alg": "FAST",
		}
		tokenClaims := map[string]interface{}{
			"client_id": "mister-client",
			"exp":       int64(3404281214),
			"iss":       "my-uaa-host",
			"scope":     []string{"notifications.write"},
		}
		tokenLoader = fakes.NewTokenLoader()

		token = fakes.BuildToken(tokenHeader, tokenClaims)
		tokenLoader.LoadCall.Returns.Token = token
		enqueuer = fakes.NewEnqueuer()
		allUsers = fakes.NewAllUsers()
		allUsers.AllUserGUIDsCall.Returns.GUIDs = []string{"user-380", "user-319"}
		strategy = services.NewEveryoneStrategy(tokenLoader, allUsers, enqueuer)
	})

	Describe("Dispatch", func() {
		It("call enqueuer.Enqueue with the correct arguments for an organization", func() {
			_, err := strategy.Dispatch(services.Dispatch{
				Connection: conn,
				Kind: services.DispatchKind{
					ID:          "welcome_user",
					Description: "Your Official Welcome",
				},
				Client: services.DispatchClient{
					ID:          "my-client",
					Description: "Welcome system",
				},
				Message: services.DispatchMessage{
					ReplyTo: "reply-to@example.com",
					Subject: "this is the subject",
					To:      "dr@strangelove.com",
					Text:    "Welcome to the system, now get off my lawn.",
					HTML: services.HTML{
						BodyContent:    "<p>Welcome to the system, now get off my lawn.</p>",
						BodyAttributes: "some-html-body-attributes",
						Head:           "<head></head>",
						Doctype:        "<html>",
					},
				},
				UAAHost: "my-uaa-host",
				VCAPRequest: services.DispatchVCAPRequest{
					ID:          "some-vcap-request-id",
					ReceiptTime: requestReceivedTime,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			var users []services.User
			for _, guid := range allUsers.AllUserGUIDsCall.Returns.GUIDs {
				users = append(users, services.User{GUID: guid})
			}

			Expect(enqueuer.EnqueueCall.Receives.Connection).To(Equal(conn))
			Expect(enqueuer.EnqueueCall.Receives.Users).To(Equal(users))
			Expect(enqueuer.EnqueueCall.Receives.Options).To(Equal(services.Options{
				ReplyTo:           "reply-to@example.com",
				Subject:           "this is the subject",
				To:                "dr@strangelove.com",
				KindID:            "welcome_user",
				KindDescription:   "Your Official Welcome",
				SourceDescription: "Welcome system",
				Text:              "Welcome to the system, now get off my lawn.",
				HTML: services.HTML{
					BodyContent:    "<p>Welcome to the system, now get off my lawn.</p>",
					BodyAttributes: "some-html-body-attributes",
					Head:           "<head></head>",
					Doctype:        "<html>",
				},
				Endorsement: services.EveryoneEndorsement,
			}))
			Expect(enqueuer.EnqueueCall.Receives.Space).To(Equal(cf.CloudControllerSpace{}))
			Expect(enqueuer.EnqueueCall.Receives.Org).To(Equal(cf.CloudControllerOrganization{}))
			Expect(enqueuer.EnqueueCall.Receives.Client).To(Equal("my-client"))
			Expect(enqueuer.EnqueueCall.Receives.Scope).To(Equal(""))
			Expect(enqueuer.EnqueueCall.Receives.VCAPRequestID).To(Equal("some-vcap-request-id"))
			Expect(enqueuer.EnqueueCall.Receives.UAAHost).To(Equal("my-uaa-host"))
			Expect(enqueuer.EnqueueCall.Receives.RequestReceived).To(Equal(requestReceivedTime))
			Expect(tokenLoader.LoadCall.Receives.UAAHost).To(Equal("my-uaa-host"))
			Expect(allUsers.AllUserGUIDsCall.Receives.Token).To(Equal(token))
		})
	})

	Context("failure cases", func() {
		Context("when token loader fails to return a token", func() {
			It("returns an error", func() {
				tokenLoader.LoadCall.Returns.Error = errors.New("BOOM!")
				_, err := strategy.Dispatch(services.Dispatch{})

				Expect(err).To(Equal(errors.New("BOOM!")))
			})
		})

		Context("when allUsers fails to load users", func() {
			It("returns the error", func() {
				allUsers.AllUserGUIDsCall.Returns.Error = errors.New("BOOM!")
				_, err := strategy.Dispatch(services.Dispatch{})

				Expect(err).To(Equal(errors.New("BOOM!")))
			})
		})
	})
})
