package main_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"strings"

	"code.google.com/p/go-uuid/uuid"
	"github.com/cloudfoundry/gorouter/token_fetcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Main", func() {
	var (
		server     *ghttp.Server
		authServer *ghttp.Server
		args       []string
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
		args = []string{
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

	Context("Given reasonable arguments", func() {

		It("registers a route to the routing api", func() {
			args = append(args, `[{"route":"zak.com","port":3,"ip":"4"}]`)

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
			args = append(args, `[{"route":"zak.com","ttl":5,"log_guid":"yo"},{"route":"jak.com","port":8,"ip":"11"}]`)
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
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("Requests a token", func() {
			args = append(args, "")
			session := routeRegistrar(args...)
			Eventually(session).Should(Exit(0))
			Expect(authServer.ReceivedRequests()).To(HaveLen(1))
			Expect(server.ReceivedRequests()).To(HaveLen(1))
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
