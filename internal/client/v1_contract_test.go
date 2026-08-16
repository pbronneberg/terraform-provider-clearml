package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newContractClient(t *testing.T, handler http.HandlerFunc) *ClearMLClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/auth.login" {
			_, _ = w.Write([]byte(`{"data":{"token":"test-token"}}`))
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	client, err := NewClearMLClient(t.Context(), "test", "key", "secret", server.URL)
	if err != nil {
		t.Fatalf("NewClearMLClient() error = %v", err)
	}
	return client
}

func TestProjectContractAndSafeDeletion(t *testing.T) {
	t.Parallel()
	step := 0
	client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.URL.Path != "/projects.create" {
				t.Fatalf("endpoint = %s, want /projects.create", r.URL.Path)
			}
			assertPayload(t, r, map[string]any{"name": "parent/project", "description": "managed", "tags": []any{"ml"}, "default_output_destination": "s3://outputs"})
			_, _ = w.Write([]byte(`{"data":{"id":"project-1"}}`))
		case 2:
			if r.URL.Path != "/projects.get_all" {
				t.Fatalf("endpoint = %s, want /projects.get_all", r.URL.Path)
			}
			assertPayload(t, r, map[string]any{"name": "^parent/project$", "page": float64(0), "page_size": float64(2)})
			_, _ = w.Write([]byte(`{"data":{"projects":[{"id":"project-1","name":"parent/project"}]}}`))
		case 3:
			if r.URL.Path != "/projects.validate_delete" {
				t.Fatalf("endpoint = %s, want /projects.validate_delete", r.URL.Path)
			}
			assertPayload(t, r, map[string]any{"project": "project-1"})
			_, _ = w.Write([]byte(`{"data":{}}`))
		case 4:
			if r.URL.Path != "/projects.delete" {
				t.Fatalf("endpoint = %s, want /projects.delete", r.URL.Path)
			}
			assertPayload(t, r, map[string]any{"project": "project-1", "force": false, "delete_contents": false, "delete_external_artifacts": false})
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			t.Fatalf("unexpected request %d to %s", step, r.URL.Path)
		}
	})
	description, destination := "managed", "s3://outputs"
	tags := []string{"ml"}
	id, err := client.CreateProject(t.Context(), ProjectInput{Name: "parent/project", Description: &description, Tags: &tags, DefaultOutputDestination: &destination})
	if err != nil || id != "project-1" {
		t.Fatalf("CreateProject() = %q, %v", id, err)
	}
	if _, err := client.FindProject(t.Context(), "parent/project"); err != nil {
		t.Fatalf("FindProject() error = %v", err)
	}
	if err := client.DeleteProject(t.Context(), id); err != nil {
		t.Fatalf("DeleteProject() error = %v", err)
	}
}

func TestProjectDeletionRefusesContents(t *testing.T) {
	t.Parallel()
	client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects.validate_delete" {
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"tasks":1}}`))
	})
	if err := client.DeleteProject(t.Context(), "project-1"); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("DeleteProject() error = %v, want non-empty refusal", err)
	}
}

func TestAccessRuleContract(t *testing.T) {
	t.Parallel()
	client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/permissions.add_or_update_access_rule" {
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		assertPayload(t, r, map[string]any{
			"name": "project-readers", "description": "least privilege", "entity_type": "project", "entity_id": "project-1",
			"access_permission": "read_write", "groups": []any{"group-1"}, "users": []any{"service-1"},
		})
		_, _ = w.Write([]byte(`{"data":{"id":"rule-1"}}`))
	})
	id, err := client.CreateAccessRule(t.Context(), AccessRuleInput{
		Name: "project-readers", Description: "least privilege", EntityType: "project", EntityID: "project-1", Permission: "read_write",
		GroupIDs: []string{"group-1"}, ServiceAccountIDs: []string{"service-1"},
	})
	if err != nil || id != "rule-1" {
		t.Fatalf("CreateAccessRule() = %q, %v", id, err)
	}
}

func TestEnterpriseNameLookupsAreExact(t *testing.T) {
	t.Parallel()
	client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users.get_service_users":
			_, _ = w.Write([]byte(`{"data":{"users":[{"id":"wrong","name":"agent-prod-extra"},{"id":"service-1","name":"agent-prod"}]}}`))
		case "/permissions.get_user_groups":
			assertPayload(t, r, map[string]any{"include_permissions": false})
			_, _ = w.Write([]byte(`{"data":{"groups":[{"id":"group-1","name":"platform"}]}}`))
		case "/resources.get_resource_configuration":
			assertPayload(t, r, map[string]any{"edit": false})
			_, _ = w.Write([]byte(`{"data":{"configuration":{"profiles":[{"id":"profile-1","name":"gpu-small"}]}}}`))
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
	})
	account, err := client.FindServiceAccount(t.Context(), "agent-prod")
	if err != nil || account.ID != "service-1" {
		t.Fatalf("FindServiceAccount() = %#v, %v", account, err)
	}
	group, err := client.FindUserGroup(t.Context(), "platform")
	if err != nil || group.ID != "group-1" {
		t.Fatalf("FindUserGroup() = %#v, %v", group, err)
	}
	profile, err := client.FindResourceProfile(t.Context(), "gpu-small")
	if err != nil || profile.ID != "profile-1" {
		t.Fatalf("FindResourceProfile() = %#v, %v", profile, err)
	}
}

func TestResourcePolicyMovePreflightRefusesQueuedWork(t *testing.T) {
	t.Parallel()
	step := 0
	client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.URL.Path != "/resources.get_policy_data" {
				t.Fatalf("unexpected endpoint %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"policy-1","user_group":"old-group"}}`))
		case 2:
			if r.URL.Path != "/resources.validate_move_policy" {
				t.Fatalf("unexpected endpoint %s", r.URL.Path)
			}
			assertPayload(t, r, map[string]any{"id": "policy-1", "user_group": "new-group"})
			_, _ = w.Write([]byte(`{"data":{"tasks_to_dequeue":2}}`))
		default:
			t.Fatalf("unsafe mutation reached %s", r.URL.Path)
		}
	})
	err := client.UpdateResourcePolicy(t.Context(), "policy-1", ResourcePolicyInput{Name: "policy", Reservation: 1, Limit: 2, UserGroupID: "new-group"})
	if err == nil || !strings.Contains(err.Error(), "dequeue") {
		t.Fatalf("UpdateResourcePolicy() error = %v, want queued-work refusal", err)
	}
}

func TestResourcePolicyDeletionRefusesDependencies(t *testing.T) {
	t.Parallel()
	for _, response := range []string{
		`{"data":{"id":"policy-1","profile_links":[{"profile":{"id":"profile-1"},"queue":{"id":"queue-1"}}]}}`,
		`{"data":{"id":"policy-1","tasks":[{"id":"task-1"}]}}`,
	} {
		response := response
		t.Run(response, func(t *testing.T) {
			client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/resources.get_policy_data" {
					t.Fatalf("unsafe mutation reached %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(response))
			})
			if err := client.DeleteResourcePolicy(t.Context(), "policy-1"); err == nil {
				t.Fatal("DeleteResourcePolicy() accepted dependent objects")
			}
		})
	}
}

func TestConnectionContractAndDisconnectPreflight(t *testing.T) {
	t.Parallel()
	displayName := "GPU queue"
	client := newContractClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/resources.connect_policy_profile":
			assertPayload(t, r, map[string]any{"policy": "policy-1", "profile": "profile-1", "queue_name": "gpu-small", "display_name": "GPU queue"})
			_, _ = w.Write([]byte(`{"data":{"id":"queue-1"}}`))
		case "/queues.get_by_id":
			assertPayload(t, r, map[string]any{"queue": "queue-1", "max_task_entries": float64(1)})
			_, _ = w.Write([]byte(`{"data":{"queue":{"id":"queue-1","entries":[{"task":"task-1"}]}}}`))
		default:
			t.Fatalf("unsafe mutation reached %s", r.URL.Path)
		}
	})
	id, err := client.ConnectResourcePolicyProfile(t.Context(), "policy-1", "profile-1", "gpu-small", &displayName)
	if err != nil || id != "queue-1" {
		t.Fatalf("ConnectResourcePolicyProfile() = %q, %v", id, err)
	}
	if err := client.DisconnectResourcePolicyProfile(t.Context(), id); err == nil || !strings.Contains(err.Error(), "contains tasks") {
		t.Fatalf("DisconnectResourcePolicyProfile() error = %v, want queued-work refusal", err)
	}
}

func TestMalformedMutationResponsesAreRejected(t *testing.T) {
	t.Parallel()
	client := newContractClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{}}`))
	})
	_, err := client.CreateResourcePolicy(t.Context(), ResourcePolicyInput{Name: "policy", UserGroupID: "group", Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "did not contain id") {
		t.Fatalf("CreateResourcePolicy() error = %v, want malformed response", err)
	}
	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		t.Fatal("malformed mutation response classified as not found")
	}
}

func TestClearMLMissingObjectSubcodeIsClassifiedAndRedacted(t *testing.T) {
	t.Parallel()
	client := newContractClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeFixture(t, w, "missing-queue.json")
	})
	_, err := client.GetQueue(t.Context(), "missing")
	if !IsNotFound(err) {
		t.Fatalf("IsNotFound(%v) = false, want true", err)
	}
	if strings.Contains(err.Error(), "must-not-leak") || strings.Contains(err.Error(), "redacted by the provider") {
		t.Fatalf("missing-object error leaked response body: %q", err)
	}
}
