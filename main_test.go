package main_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"

	"os"

	"github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/db"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	uuid "github.com/nu7hatch/gouuid"
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
			tokenUuid, err := uuid.NewV4()
			Expect(err).NotTo(HaveOccurred())
			token = tokenUuid.String()
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
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/routing/v1/tcp_routes/routes"),
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
			var (
				httpEvent       routing_api.Event
				tcpEvent        routing_api.TcpEvent
				httpEventString []byte
				tcpEventString  []byte
				sseEvent        sse.Event
				sseEventTcp     sse.Event
				headers         http.Header
			)

			BeforeEach(func() {

				httpEvent = routing_api.Event{
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

				tcpEvent = routing_api.TcpEvent{
					Action: "Upsert",
					TcpRouteMapping: db.TcpRouteMapping{
						TcpRoute: db.TcpRoute{
							RouterGroupGuid: "some-guid",
							ExternalPort:    1234,
						},
						HostPort: 6789,
						HostIP:   "some-ip",
					},
				}

				var err error
				httpEventString, err = json.Marshal(httpEvent.Route)
				Expect(err).ToNot(HaveOccurred())

				sseEvent = sse.Event{
					Name: httpEvent.Action,
					Data: httpEventString,
				}

				headers = make(http.Header)
				headers.Set("Content-Type", "text/event-stream; charset=utf-8")

				server.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/routing/v1/events"),
						ghttp.RespondWith(http.StatusOK, sseEvent.Encode(), headers),
					),
				)

				tcpEventString, err = json.Marshal(tcpEvent.TcpRouteMapping)
				Expect(err).ToNot(HaveOccurred())

				sseEventTcp = sse.Event{
					Name: tcpEvent.Action,
					Data: tcpEventString,
				}

			})

			It("emits an error message on server termination", func() {
				command := buildCommand("events", flags, []string{})

				server.SetHandler(0,
					ghttp.CombineHandlers(
						ghttp.RespondWith(http.StatusOK, ""),
					),
				)
				server.SetHandler(1,
					ghttp.CombineHandlers(
						ghttp.RespondWith(http.StatusOK, ""),
					),
				)

				session := routeRegistrar(command...)

				Eventually(session, "2s").Should(Exit(0))
				Expect(server.ReceivedRequests()).To(HaveLen(2))
				Expect(string(session.Out.Contents())).To(ContainSubstring("Connection closed: "))
			})

			Context("when --http flag is provided", func() {
				var flagsWithHttp []string

				BeforeEach(func() {
					flagsWithHttp = append(flags, "--http")
				})

				It("subscribes to HTTP events", func() {
					command := buildCommand("events", flagsWithHttp, []string{})

					session := routeRegistrar(command...)

					Eventually(session, "2s").Should(Exit(0))
					Expect(server.ReceivedRequests()).To(HaveLen(1))
					Expect(string(session.Out.Contents())).To(ContainSubstring(string(httpEventString)))
					Expect(string(session.Out.Contents())).NotTo(ContainSubstring(string(tcpEventString)))
				})
			})

			Context("when --tcp flag is provided", func() {
				var flagsWithTcp []string

				BeforeEach(func() {
					server.SetHandler(0,
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/routing/v1/tcp_routes/events"),
							ghttp.RespondWith(http.StatusOK, sseEventTcp.Encode(), headers),
						),
					)
					flagsWithTcp = append(flags, "--tcp")
				})

				It("subscribes to TCP events", func() {
					command := buildCommand("events", flagsWithTcp, []string{})

					session := routeRegistrar(command...)

					Eventually(session, "2s").Should(Exit(0))
					Expect(server.ReceivedRequests()).To(HaveLen(1))
					Expect(string(session.Out.Contents())).To(ContainSubstring(string(tcpEventString)))
					Expect(string(session.Out.Contents())).NotTo(ContainSubstring(string(httpEventString)))
				})
			})

			Context("when both --http and --tcp flags are provided", func() {
				var flagsWithAllProtocols []string

				BeforeEach(func() {
					eventHandler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)

						if req.URL.Path == "/routing/v1/events" {
							w.Write([]byte(sseEvent.Encode()))
						} else {
							w.Write([]byte(sseEventTcp.Encode()))
						}
					}

					server.SetHandler(0, eventHandler)
					server.SetHandler(1, eventHandler)

					flagsWithAllProtocols = append(flags, "--http", "--tcp")
				})

				It("subscribes to HTTP and TCP events", func() {
					command := buildCommand("events", flagsWithAllProtocols, []string{})

					session := routeRegistrar(command...)

					Eventually(session, "2s").Should(Exit(0))
					Expect(server.ReceivedRequests()).To(HaveLen(2))
					Expect(string(session.Out.Contents())).To(ContainSubstring(string(tcpEventString)))
					Expect(string(session.Out.Contents())).To(ContainSubstring(string(httpEventString)))
				})
			})

			Context("when no protocol specific flag is provided", func() {
				BeforeEach(func() {
					eventHandler := func(w http.ResponseWriter, req *http.Request) {
						w.WriteHeader(http.StatusOK)

						if req.URL.Path == "/routing/v1/events" {
							w.Write([]byte(sseEvent.Encode()))
						} else {
							w.Write([]byte(sseEventTcp.Encode()))
						}
					}

					server.SetHandler(0, eventHandler)
					server.SetHandler(1, eventHandler)
				})
				It("subscribes to HTTP and TCP events", func() {
					command := buildCommand("events", flags, []string{})

					session := routeRegistrar(command...)

					Eventually(session, "2s").Should(Exit(0))
					Expect(server.ReceivedRequests()).To(HaveLen(2))
					Expect(string(session.Out.Contents())).To(ContainSubstring(string(tcpEventString)))
					Expect(string(session.Out.Contents())).To(ContainSubstring(string(httpEventString)))
				})
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
				Eventually(session).Should(Say("Error fetching oauth token:"))
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
