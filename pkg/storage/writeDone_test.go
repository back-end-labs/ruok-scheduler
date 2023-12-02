package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/back-end-labs/ruok/pkg/config"
	"github.com/back-end-labs/ruok/pkg/job"
)

var seedWriteDoneQuery = `
INSERT INTO jobs (
	id,
	cron_exp_string,
	endpoint,
	httpmethod,
	max_retries,
	success_statuses,
	status,
	claimed_by
) VALUES (1,'* * * * *', '/', 'GET', 1, '{200}',  'claimed','application1')
`

var makeJobStruct = func(now time.Time) job.Job {
	return job.Job{
		Id:              1,
		CronExpString:   "* * * * *",
		LastExecution:   now,
		ShouldExecuteAt: now,
		LastResponseAt:  now,
		Status:          "claimed",
		ClaimedBy:       config.AppName(),
		LastStatusCode:  200,
		LastMessage:     "OK",
		HttpMethod:      "GET",
		Endpoint:        "/",
		SuccessStatuses: []int{200},
		Headers:         []job.Header{},
		MaxRetries:      1,
	}
}

var selectJobExecutionQuery = `
SELECT 
	id,
	job_id,
	cron_exp_string,
	endpoint,
	httpmethod,
	max_retries,
	execution_time,
	should_execute_at,
	last_response_at,
	last_message,
	last_status_code,
	success_statuses,
	claimed_by
 FROM job_results 
WHERE job_id = $1
`

var queryDoneJob = `
SELECT
	last_execution,
	should_execute_at,
	last_response_at,
	last_message,
	last_status_code
FROM jobs
WHERE id = $1
`

func TestWriteDone(t *testing.T) {
	Drop()
	defer Drop()
	t.Run("Done jobs are written as they should", func(t *testing.T) {
		cfg := config.FromEnvs()
		s, close := NewStorage(&cfg)
		defer close()

		ctx := context.Background()

		_, err := s.GetClient().Exec(ctx, seedWriteDoneQuery)

		if err != nil {
			t.Errorf("couln't seed due to the following error: %q", err.Error())
		}

		now := time.Now()

		j := makeJobStruct(now)

		err = s.WriteDone(&j)

		if err != nil {
			t.Errorf("wrriting a job result shouldn't error. error=%q\n", err.Error())
		}

		// Clousure for job_execution asserts
		{
			ctx = context.Background()

			var (
				id              int64
				jobID           int64
				cronExpString   string
				endpoint        string
				httpMethod      string
				maxRetries      int
				executionTime   int64
				shouldExecuteAt int64
				lastResponseAt  int64
				lastMessage     sql.NullString
				lastStatusCode  int
				successStatuses []int
				tlsClientCert   sql.NullString
				claimedBy       string
			)
			row := s.GetClient().QueryRow(ctx, selectJobExecutionQuery, j.Id)
			err = row.Scan(
				&id,
				&jobID,
				&cronExpString,
				&endpoint,
				&httpMethod,
				&maxRetries,
				&executionTime,
				&shouldExecuteAt,
				&lastResponseAt,
				&lastMessage,
				&lastStatusCode,
				&successStatuses,
				&claimedBy,
			)
			if err != nil {
				t.Errorf("querying a done job should not produce an error. error=%q\n", err.Error())
			}

			checkJobExecutionFields(
				jobID,
				j,
				t,
				cronExpString,
				endpoint,
				httpMethod,
				maxRetries,
				executionTime,
				shouldExecuteAt,
				lastResponseAt,
				lastMessage,
				lastStatusCode,
				tlsClientCert,
				claimedBy)

		}

		// clousure for jobs asserts
		{

			var executionTime int64
			var shouldExecuteAt int64
			var lastResponseAt int64
			var lastMessage sql.NullString
			var lastStatusCode int

			ctx := context.Background()

			row := s.GetClient().QueryRow(ctx, queryDoneJob, j.Id)

			err := row.Scan(
				&executionTime,
				&shouldExecuteAt,
				&lastResponseAt,
				&lastMessage,
				&lastStatusCode,
			)

			if err != nil {
				t.Errorf("couldn't get job after updating it: %q", err.Error())
			}

			checkDoneJobFields(executionTime, j, t, shouldExecuteAt, lastResponseAt, lastMessage, lastStatusCode)

		}

	})

}

func checkDoneJobFields(
	executionTime int64,
	j job.Job,
	t *testing.T,
	shouldExecuteAt int64,
	lastResponseAt int64,
	lastMessage sql.NullString,
	lastStatusCode int,
) {
	if executionTime != j.LastExecution.UnixMicro() {
		t.Errorf("Expected ExecutionTime: %v, Got: %v", j.LastExecution.UnixMicro(), executionTime)
	}

	if shouldExecuteAt != j.ShouldExecuteAt.UnixMicro() {
		t.Errorf("Expected ShouldExecuteAt: %v, Got: %v", j.ShouldExecuteAt.UnixMicro(), shouldExecuteAt)
	}

	if lastResponseAt != j.LastResponseAt.UnixMicro() {
		t.Errorf("Expected LastResponseAt: %v, Got: %v", j.LastResponseAt.UnixMicro(), lastResponseAt)
	}

	if lastMessage.String != j.LastMessage {
		t.Errorf("Expected LastMessage: %s, Got: %s", j.LastMessage, lastMessage.String)
	}

	if lastStatusCode != j.LastStatusCode {
		t.Errorf("Expected LastStatusCode: %d, Got: %d", j.LastStatusCode, lastStatusCode)
	}
}

func checkJobExecutionFields(
	jobID int64,
	j job.Job,
	t *testing.T,
	cronExpString string,
	endpoint string,
	httpMethod string,
	maxRetries int,
	executionTime int64,
	shouldExecuteAt int64,
	lastResponseAt int64,
	lastMessage sql.NullString,
	lastStatusCode int,
	tlsClientCert sql.NullString,
	claimedBy string,
) {
	if jobID != int64(j.Id) {
		t.Errorf("Expected JobID: %d, Got: %d", j.Id, jobID)
	}

	if cronExpString != j.CronExpString {
		t.Errorf("Expected CronExpString: %s, Got: %s", j.CronExpString, cronExpString)
	}

	if endpoint != j.Endpoint {
		t.Errorf("Expected Endpoint: %s, Got: %s", j.Endpoint, endpoint)
	}

	if httpMethod != j.HttpMethod {
		t.Errorf("Expected HttpMethod: %s, Got: %s", j.HttpMethod, httpMethod)
	}

	if maxRetries != j.MaxRetries {
		t.Errorf("Expected MaxRetries: %d, Got: %d", j.MaxRetries, maxRetries)
	}

	if executionTime != j.LastExecution.UnixMicro() {
		t.Errorf("Expected ExecutionTime: %d, Got: %d", j.LastExecution.UnixMicro(), executionTime)
	}

	if shouldExecuteAt != j.ShouldExecuteAt.UnixMicro() {
		t.Errorf("Expected ShouldExecuteAt: %d, Got: %d", j.ShouldExecuteAt.UnixMicro(), shouldExecuteAt)
	}

	if lastResponseAt != j.LastResponseAt.UnixMicro() {
		t.Errorf("Expected LastResponseAt: %d, Got: %d", j.LastResponseAt.UnixMicro(), lastResponseAt)
	}

	if lastMessage.String != j.LastMessage {
		t.Errorf("Expected LastMessage: %s, Got: %s", j.LastMessage, lastMessage.String)
	}

	if lastStatusCode != j.LastStatusCode {
		t.Errorf("Expected LastStatusCode: %d, Got: %d", j.LastStatusCode, lastStatusCode)
	}

	if tlsClientCert.String != j.TLSClientCert {
		t.Errorf("Expected TlsClientCert: %s, Got: %s", j.TLSClientCert, tlsClientCert.String)
	}

	if claimedBy != j.ClaimedBy {
		t.Errorf("Expected ClaimedBy: %s, Got: %s", j.ClaimedBy, claimedBy)
	}
}
