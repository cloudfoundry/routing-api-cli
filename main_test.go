package main_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"

	"os"

	"code.google.com/p/go-uuid/uuid"
	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/vito/go-sse/sse"
)

var _ = Describe("Main", func() {
	var (
		flags []string
	)

	var buildCommand = func(cmd string, flags []string, args []string) []string {
		command := []string{cmd}
		command = append(command, flags...)
		command = append(command, args...)
		return command
	}

	Context("Given reasonable arguments", func() {
		var (
			server     *ghttp.Server
			authServer *ghttp.Server
			token      string
		)

		BeforeEach(func() {
			server = ghttp.NewServer()
			authServer = ghttp.NewServer()
			token = uuid.NewUUID().String()
			responseBody := &token_fetcher.Token{
				AccessToken: token,
				ExpireTime:  20,
			}

			authServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.VerifyBasicAuth("some-name", "some-secret"),
					ghttp.VerifyContentType("application/x-www-form-urlencoded; charset=UTF-8"),
					ghttp.VerifyHeader(http.Header{
						"Accept": []string{"application/json; charset=utf-8"},
					}),
					verifyBody("grant_type=client_credentials"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, responseBody),
				))

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/routing/v1/routes"),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{"bearer " + token},
					}),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			flags = []string{
				"-api", server.URL(),
				"-client-id", "some-name",
				"-client-secret", "some-secret",
				"-oauth-url", authServer.URL(),
			}
		})

		AfterEach(func() {
			authServer.Close()
			server.Close()
		})

		It("registers a route to the routing api", func() {
			command := buildCommand("register", flags, []string{`[{"route":"zak.com","port":3,"ip":"4"}]`})

			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/routing/v1/routes"),
					ghttp.VerifyJSONRepresenting([]map[string]interface{}{
						{
							"route":    "zak.com",
							"port":     3,
							"ip":       "4",
							"ttl":      0,
							"log_guid": "",
						},
					}),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			session := routeRegistrar(command...)

			Eventually(session, "2s").Should(Exit(0))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("registers multiple routes to the routing api", func() {
			routes := `[{"route":"zak.com","port":0,"ip": "","ttl":5,"log_guid":"yo"},{"route":"jak.com","port":8,"ip":"11","ttl":0}]`
			command := buildCommand("register", flags, []string{routes})
			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/routing/v1/routes"),
					ghttp.VerifyJSONRepresenting([]map[string]interface{}{
						{
							"route":    "zak.com",
							"port":     0,
							"ip":       "",
							"ttl":      5,
							"log_guid": "yo",
						},
						{
							"route":    "jak.com",
							"port":     8,
							"ip":       "11",
							"ttl":      0,
							"log_guid": "",
						},
					}),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			session := routeRegistrar(command...)

			Eventually(session, "2s").Should(Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("Successfully registered routes: " + routes + "\n"))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("Unregisters a route to the routing api", func() {
			routes := `[{"route":"zak.com","ttl":5,"log_guid":"yo"}]`
			command := buildCommand("unregister", flags, []string{routes})

			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/routing/v1/routes"),
					ghttp.VerifyJSONRepresenting([]map[string]interface{}{
						{
							"route":    "zak.com",
							"port":     0,
							"ip":       "",
							"ttl":      5,
							"log_guid": "yo",
						},
					}),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			session := routeRegistrar(command...)

			Eventually(session, "2s").Should(Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("Successfully unregistered routes: " + routes))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("Lists the routes", func() {
			routes := []db.Route{
				{Route: "llama.example.com", Port: 0, IP: "", TTL: 5, LogGuid: "yo"},
				{Route: "example.com", Port: 8, IP: "11", TTL: 0},
			}
			command := buildCommand("list", flags, []string{})

			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/routing/v1/routes"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, routes),
				),
			)

			session := routeRegistrar(command...)

			expectedRoutes, err := json.Marshal(routes)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session, "2s").Should(Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring(string(expectedRoutes) + "\n"))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		Context("events", func() {
			It("subscribes to routing API events", func() {
				event := routing_api.Event{
					Action: "Delete",
					Route: db.Route{
						Route:           "z.a.k",
						Port:            63,
						IP:              "42.42.42.42",
						TTL:             1,
						LogGuid:         "Tomato",
						RouteServiceUrl: "https://route-service-url.com",
					},
				}

				routeString, err := json.Marshal(event.Route)
				Expect(err).ToNot(HaveOccurred())

				sseEvent := sse.Event{
					Name: event.Action,
					Data: routeString,
				}

				headers := make(http.Header)
				headers.Set("Content-Type", "text/event-stream; charset=utf-8")

				command := buildCommand("events", flags, []string{})

				server.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routing/v1/events"),
						ghttp.RespondWith(http.StatusOK, sseEvent.Encode(), headers),
					),
				)

				session := routeRegistrar(command...)

				eventString, err := json.Marshal(event)
				Expect(err).ToNot(HaveOccurred())
				eventString = append(eventString, '\n')

				Eventually(session, "2s").Should(Exit(0))
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(string(session.Out.Contents())).To(ContainSubstring(string(eventString)))
			})

			It("emits an error message on server termination", func() {
				command := buildCommand("events", flags, []string{})

				server.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routing/v1/events"),
						ghttp.RespondWith(http.StatusOK, ""),
					),
				)

				session := routeRegistrar(command...)

				Eventually(session, "2s").Should(Exit(0))
				Expect(server.ReceivedRequests()).To(HaveLen(1))
				Expect(string(session.Err.Contents())).To(ContainSubstring("Connection closed: "))
			})
		})

		It("Requests a token", func() {
			command := buildCommand("register", flags, []string{"[{}]"})
			session := routeRegistrar(command...)

			Eventually(session, "2s").Should(Exit(0))
			Expect(authServer.ReceivedRequests()).To(HaveLen(1))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		Context("environment variables", func() {
			Context("RTR_TRACE", func() {
				var session *Session
				BeforeEach(func() {
					routes := []db.Route{
						{Route: "llama.example.com", Port: 0, IP: "", TTL: 5, LogGuid: "yo"},
						{Route: "example.com", Port: 8, IP: "11", TTL: 0},
					}
					server.SetHandler(0,
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/routing/v1/routes"),
							ghttp.RespondWithJSONEncoded(http.StatusOK, routes),
						),
					)
				})

				JustBeforeEach(func() {
					command := buildCommand("list", flags, []string{})
					session = routeRegistrar(command...)
					Eventually(session, "2s").Should(Exit(0))
				})

				Context("when RTR_TRACE is not set", func() {
					BeforeEach(func() {
						os.Unsetenv("RTR_TRACE")
					})

					It("should not trace the requests made/responses received", func() {
						Expect(string(session.Out.Contents())).NotTo(ContainSubstring("REQUEST"))
					})
				})

				Context("when RTR_TRACE is set to true", func() {
					BeforeEach(func() {
						os.Setenv("RTR_TRACE", "true")
					})

					It("should trace the requests made/responses received", func() {
						Expect(string(session.Out.Contents())).To(ContainSubstring("REQUEST"))
					})
				})

				Context("when RTR_TRACE is set to false", func() {
					BeforeEach(func() {
						os.Setenv("RTR_TRACE", "false")
					})

					It("should not trace the requests made/responses received", func() {
						Expect(string(session.Out.Contents())).NotTo(ContainSubstring("REQUEST"))
					})
				})

				Context("when RTR_TRACE is set to an invalid value", func() {
					BeforeEach(func() {
						os.Setenv("RTR_TRACE", "adsf")
					})

					It("should not trace the requests made/responses received", func() {
						Expect(string(session.Out.Contents())).NotTo(ContainSubstring("REQUEST"))
					})
				})
			})
		})
	})

	Context("Given unreasonable arguments", func() {
		BeforeEach(func() {
			flags = []string{
				"-api", "some-server-name",
				"-client-id", "some-name",
				"-client-secret", "some-secret",
				"-oauth-url", "http://some.oauth.url",
			}
		})

		Context("when no API endpoint is specified", func() {
			BeforeEach(func() {
				flags = []string{
					"-client-id", "some-name",
					"-client-secret", "some-secret",
					"-oauth-url", "http://some.oauth.url",
				}
			})

			It("checks for the presence of api", func() {
				command := buildCommand("register", []string{}, []string{})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Must provide an API endpoint for the routing-api component.\n"))
			})
		})

		Context("when no flags are given", func() {
			It("tells you everything you did wrong", func() {
				session := routeRegistrar("register")

				Eventually(session).Should(Exit(1))
				contents := session.Out.Contents()
				Expect(contents).To(ContainSubstring("Must provide an API endpoint for the routing-api component.\n"))
				Expect(contents).To(ContainSubstring("Must provide the id of an OAuth client.\n"))
				Expect(contents).To(ContainSubstring("Must provide an OAuth secret.\n"))
				Expect(contents).To(ContainSubstring("Must provide an URL to the OAuth client.\n"))
			})
		})

		It("checks for a valid command", func() {
			session := routeRegistrar("not-a-command")

			Eventually(session).Should(Exit(1))
			Eventually(session).Should(Say("Not a valid command: not-a-command"))
		})

		It("outputs help info for a valid command", func() {
			session := routeRegistrar("register")

			Eventually(session).Should(Exit(1))
			Eventually(session).Should(Say("command register"))
		})

		It("outputs help info for a valid command", func() {
			session := routeRegistrar("events")

			Eventually(session).Should(Exit(1))
			Eventually(session).Should(Say("command events"))
		})

		It("outputs help info for a valid command", func() {
			session := routeRegistrar("unregister")

			Eventually(session).Should(Exit(1))
			Eventually(session).Should(Say("command unregister"))
		})

		Context("register", func() {
			It("checks for the presence of the route json", func() {
				command := buildCommand("register", flags, []string{})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Must provide routes JSON."))
			})

			It("fails if the request has invalid json", func() {
				command := buildCommand("register", flags, []string{`[{"kind":"of","valid":"json}]`})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("Invalid json format."))
			})

			It("fails if there are unexpected arguments", func() {
				command := buildCommand("register", flags, []string{`[{"kind":"of","valid":"json}]`, "ice cream"})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Unexpected arguments."))
			})

			It("shows the error if registration fails", func() {
				command := buildCommand("register", flags, []string{"[{}]"})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("route registration failed:"))
			})
		})

		Context("unregister", func() {
			It("checks for the presence of the route json", func() {
				command := buildCommand("unregister", flags, []string{})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Must provide routes JSON."))
			})

			It("fails if the unregister request has invalid json", func() {
				command := buildCommand("unregister", flags, []string{`[{"kind":"of","valid":"json}]`})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("Invalid json format."))
			})

			It("fails if there are unexpected arguments", func() {
				command := buildCommand("unregister", flags, []string{`[{"kind":"of","valid":"json}]`, "ice cream"})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Unexpected arguments."))
			})

			It("shows the error if unregistration fails", func() {
				command := buildCommand("unregister", flags, []string{"[{}]"})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("route unregistration failed:"))
			})
		})

		Context("events", func() {
			It("fails if there are unexpected arguments", func() {
				command := buildCommand("events", flags, []string{"ice cream"})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Unexpected arguments."))
			})

			It("shows the error if streaming events fails", func() {
				command := buildCommand("events", flags, []string{})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("streaming events failed:"))
			})
		})

		Context("list", func() {
			It("fails if there are unexpected arguments", func() {
				command := buildCommand("list", flags, []string{"ice cream"})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(1))
				Eventually(session).Should(Say("Unexpected arguments."))
			})

			It("shows the error if listing routes fails", func() {
				command := buildCommand("list", flags, []string{})
				session := routeRegistrar(command...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("listing routes failed:"))
			})
		})
	})
})

func routeRegistrar(args ...string) *Session {
	session, err := Start(exec.Command(path, args...), GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}

func verifyBody(expectedBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		Expect(err).ToNot(HaveOccurred())

		defer r.Body.Close()
		Expect(string(body)).To(Equal(expectedBody))
	}
}
