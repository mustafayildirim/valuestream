// +build service

package gitlab

import (
	"encoding/json"
	"fmt"
	"github.com/ImpactInsights/valuestream/eventsources"
	"github.com/ImpactInsights/valuestream/tracers"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"
)

var baseURL string
var gitlabPath string

var urlEnvVar string = "TEST_EVENTS_URL"
var gitlabPathEnvVar string = "TEST_EVENTS_GITLAB_PATH"

func init() {
	ok := true
	baseURL, ok = os.LookupEnv(urlEnvVar)
	if !ok {
		panic(fmt.Sprintf("requires: %q", urlEnvVar))
	}
	gitlabPath, ok = os.LookupEnv(gitlabPathEnvVar)
	if !ok {
		panic(fmt.Sprintf("requires: %q", urlEnvVar))
	}
}

var eventTests = []struct {
	Name                  string
	StartEventPath        string
	EndEventPath          string
	ExpectedOperationName string
	ExpectedTags          map[string]interface{}
}{
	{
		Name:                  "pipeline_pending_running",
		StartEventPath:        "fixtures/events/pipeline/pending.json",
		EndEventPath:          "fixtures/events/pipeline/running.json",
		ExpectedOperationName: "pipeline",
		ExpectedTags: map[string]interface{}{
			"project.name":                "test-project",
			"project.url":                 "https://gitlab.com/dm03514/test-project",
			"vstrace.state":               "start",
			"build.ref":                   "feature/test",
			"event.type":                  "pipeline",
			"build.tag":                   false,
			"event.state":                 "pending",
			"project.id":                  float64(1.5119184e+07),
			"vstrace.span.id":             "vstrace-gitlab-build-test-project-96963426",
			"build.id":                    float64(9.6963426e+07),
			"build.sha":                   "304839c04c12d78a94b9b521c237c83ec84e826d",
			"user.username":               "dm03514",
			"project.path_with_namespace": "dm03514/test-project",
			"project.visibility":          "",
			"project.namespace":           "Daniel Mican",
			"service":                     "gitlab",
			"user.name":                   "Daniel Mican",
			"build.before_sha":            "0000000000000000000000000000000000000000",
			"error":                       false,
		},
	},
	{
		Name:                  "pipeline_running_success",
		StartEventPath:        "fixtures/events/pipeline/running.json",
		EndEventPath:          "fixtures/events/pipeline/success.json",
		ExpectedOperationName: "pipeline",
		ExpectedTags: map[string]interface{}{
			"project.name":                "test-project",
			"project.url":                 "https://gitlab.com/dm03514/test-project",
			"vstrace.state":               "transition",
			"build.ref":                   "feature/test",
			"event.type":                  "pipeline",
			"build.tag":                   false,
			"event.state":                 "running",
			"project.id":                  float64(1.5119184e+07),
			"vstrace.span.id":             "vstrace-gitlab-build-test-project-96963426",
			"build.id":                    float64(9.6963426e+07),
			"build.sha":                   "304839c04c12d78a94b9b521c237c83ec84e826d",
			"user.username":               "dm03514",
			"project.path_with_namespace": "dm03514/test-project",
			"project.visibility":          "",
			"project.namespace":           "Daniel Mican",
			"service":                     "gitlab",
			"user.name":                   "Daniel Mican",
			"build.before_sha":            "0000000000000000000000000000000000000000",
			"error":                       false,
		},
	},
}

func TestServiceEvent_Gitlab(t *testing.T) {
	client := &http.Client{}
	u, err := url.Parse(baseURL + gitlabPath)
	assert.NoError(t, err)

	var te *eventsources.TestEvent

	for _, tt := range eventTests {
		t.Run(tt.Name, func(t *testing.T) {
			// reset the tracer
			resp, err := http.Get(baseURL + "/mocktracer/reset")
			assert.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			eventPaths := []string{
				tt.StartEventPath,
				tt.EndEventPath,
			}
			for _, eventPath := range eventPaths {
				te, err = eventsources.NewTestEventFromFixturePath(eventPath)

				rawPayload, err := json.Marshal(te.Payload)
				assert.NoError(t, err)

				eventResp, err := PostEvent(
					rawPayload,
					te.Headers["X-Gitlab-Event"],
					u,
					client,
				)

				assert.NoError(t, err)
				eventResp.Body.Close()
				assert.Equal(t, http.StatusOK, eventResp.StatusCode)
			}

			spansResp, err := http.Get(baseURL + "/mocktracer/finished-spans")
			assert.NoError(t, err)

			bs, err := ioutil.ReadAll(spansResp.Body)
			assert.NoError(t, err)
			spansResp.Body.Close()

			fmt.Println(string(bs))

			assert.Equal(t, http.StatusOK, spansResp.StatusCode)

			var spans []tracers.TestSpan

			err = json.Unmarshal(bs, &spans)
			assert.NoError(t, err)

			for k, v := range spans[0].Tags {
				fmt.Printf("%q:%q\n", k, v)
			}

			assert.Equal(t, 1, len(spans))
			assert.Equal(t, tt.ExpectedOperationName, spans[0].Span.OperationName)
			assert.Equal(t, tt.ExpectedTags, spans[0].Tags)
		})
	}
}
