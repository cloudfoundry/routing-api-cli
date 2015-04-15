package main_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"code.google.com/p/go-uuid/uuid"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Main", func() {
	var (
		flags []string
	)

	var buildArgs = func(cmd string, json ...string) []string {
		args := []string{cmd}
		args = append(args, flags...)
		return append(args, json...)
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
					ghttp.VerifyBasicAuth("some-name", "some-password"),
					ghttp.VerifyContentType("application/x-www-form-urlencoded; charset=UTF-8"),
					ghttp.VerifyHeader(http.Header{
						"Accept": []string{"application/json; charset=utf-8"},
					}),
					verifyBody("grant_type=client_credentials"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, responseBody),
				))

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v1/routes"),
					ghttp.VerifyHeader(http.Header{
						"Authorization": []string{"bearer " + token},
					}),
					ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
				),
			)

			url, err := url.Parse(authServer.URL())
			Expect(err).ToNot(HaveOccurred())

			addr := strings.Split(url.Host, ":")
			flags = []string{
				"-api", server.URL(),
				"-oauth-name", "some-name",
				"-oauth-password", "some-password",
				"-oauth-url", "http://" + addr[0],
				"-oauth-port", addr[1],
			}
		})

		AfterEach(func() {
			authServer.Close()
			server.Close()
		})

		It("registers a route to the routing api", func() {
			args := buildArgs("register", `[{"route":"zak.com","port":3,"ip":"4"}]`)

			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v1/routes"),
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

			session := routeRegistrar(args...)

			Eventually(session).Should(Exit(0))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("registers multiple route to the routing api", func() {
			routes := `[{"route":"zak.com","port":0,"ip": "","ttl":5,"log_guid":"yo"},{"route":"jak.com","port":8,"ip":"11","ttl":0}]`
			args := buildArgs("register", routes)
			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/v1/routes"),
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

			session := routeRegistrar(args...)

			Eventually(session).Should(Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("Successfuly registered routes: " + routes))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("Unregisters a route to the routing api", func() {
			routes := `[{"route":"zak.com","ttl":5,"log_guid":"yo"}]`
			args := buildArgs("unregister", routes)

			server.SetHandler(0,
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/v1/routes"),
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

			session := routeRegistrar(args...)

			Eventually(session).Should(Exit(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("Successfuly unregistered routes: " + routes))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("Requests a token", func() {
			args := buildArgs("register", "[{}]")
			session := routeRegistrar(args...)

			Eventually(session).Should(Exit(0))
			Expect(authServer.ReceivedRequests()).To(HaveLen(1))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})
	})

	Context("Given unreasonable arguments", func() {
		BeforeEach(func() {
			flags = []string{
				"-api", "some-server-name",
				"-oauth-name", "some-name",
				"-oauth-password", "some-password",
				"-oauth-url", "http://some.oauth.url",
				"-oauth-port", "666",
			}
		})

		Context("when no API endpoint is specified", func() {
			BeforeEach(func() {
				flags = []string{
					"-oauth-name", "some-name",
					"-oauth-password", "some-password",
					"-oauth-url", "http://some.oauth.url",
					"-oauth-port", "666",
				}
			})

			It("checks for the presence of api", func() {
				args := buildArgs("register", "")
				session := routeRegistrar(args...)

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
				Expect(contents).To(ContainSubstring("Must provide the name of an OAuth client.\n"))
				Expect(contents).To(ContainSubstring("Must provide an OAuth password/secret.\n"))
				Expect(contents).To(ContainSubstring("Must provide an URL to the OAuth client.\n"))
				Expect(contents).To(ContainSubstring("Must provide the port the OAuth client is listening on.\n"))
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
			session := routeRegistrar("unregister")

			Eventually(session).Should(Exit(1))
			Eventually(session).Should(Say("command unregister"))
		})

		It("checks for the presence of the route json", func() {
			args := []string{"register"}
			args = append(args, flags...)
			session := routeRegistrar(args...)

			Eventually(session).Should(Exit(1))
			Eventually(session).Should(Say("Must provide routes JSON."))
		})

		Context("register", func() {
			It("fails if the request has invalid json", func() {
				args := buildArgs("register", `[{"kind":"of","valid":"json}]`)
				args = append(args, flags...)
				session := routeRegistrar(args...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("Invalid json format."))
			})

			It("shows the error if registration fails", func() {
				args := buildArgs("register", "[{}]")
				session := routeRegistrar(args...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("route registration failed:"))
			})
		})

		Context("unregister", func() {
			It("fails if the unregister request has invalid json", func() {
				args := buildArgs("unregister", `[{"kind":"of","valid":"json}]`)
				args = append(args, flags...)
				session := routeRegistrar(args...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("Invalid json format."))
			})

			It("shows the error if unregistration fails", func() {
				args := buildArgs("unregister", "[{}]")
				session := routeRegistrar(args...)

				Eventually(session).Should(Exit(3))
				Eventually(session).Should(Say("route unregistration failed:"))
			})
		})
	})
})

func routeRegistrar(args ...string) *Session {
	path, err := Build("github.com/cloudfoundry-incubator/routing-api-cli")
	Expect(err).NotTo(HaveOccurred())

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
