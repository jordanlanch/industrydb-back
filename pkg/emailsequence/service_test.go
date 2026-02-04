package emailsequence

import (
	"context"
	"testing"
	"time"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:"+t.Name()+"?mode=memory&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	now := time.Now()
	user, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed").
		SetName(name).
		SetEmailVerifiedAt(now).
		Save(context.Background())
	require.NoError(t, err)
	return user
}

func createTestLead(t *testing.T, client *ent.Client, name, email string) *ent.Lead {
	lead, err := client.Lead.
		Create().
		SetName(name).
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("NYC").
		SetEmail(email).
		Save(context.Background())
	require.NoError(t, err)
	return lead
}

func TestCreateSequence(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")

	t.Run("Success - Create sequence", func(t *testing.T) {
		req := CreateSequenceRequest{
			Name:        "Welcome Series",
			Description: "Onboarding emails for new leads",
			Trigger:     "manual",
		}

		result, err := service.CreateSequence(ctx, user.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "Welcome Series", result.Name)
		assert.Equal(t, "Onboarding emails for new leads", result.Description)
		assert.Equal(t, "draft", result.Status)
		assert.Equal(t, "manual", result.Trigger)
		assert.Equal(t, user.ID, result.CreatedBy)
	})
}

func TestGetSequence(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")

	req := CreateSequenceRequest{
		Name:    "Test Sequence",
		Trigger: "manual",
	}
	sequence, err := service.CreateSequence(ctx, user.ID, req)
	require.NoError(t, err)

	t.Run("Success - Get sequence", func(t *testing.T) {
		result, err := service.GetSequence(ctx, sequence.ID)

		require.NoError(t, err)
		assert.Equal(t, sequence.ID, result.ID)
		assert.Equal(t, "Test Sequence", result.Name)
	})

	t.Run("Error - Sequence not found", func(t *testing.T) {
		result, err := service.GetSequence(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestListSequences(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@test.com", "User 1")
	user2 := createTestUser(t, client, "user2@test.com", "User 2")

	// Create sequences for user1
	service.CreateSequence(ctx, user1.ID, CreateSequenceRequest{Name: "Seq 1", Trigger: "manual"})
	service.CreateSequence(ctx, user1.ID, CreateSequenceRequest{Name: "Seq 2", Trigger: "manual"})

	// Create sequence for user2
	service.CreateSequence(ctx, user2.ID, CreateSequenceRequest{Name: "Seq 3", Trigger: "manual"})

	t.Run("Success - List sequences for user", func(t *testing.T) {
		sequences, err := service.ListSequences(ctx, user1.ID)

		require.NoError(t, err)
		assert.Len(t, sequences, 2)
	})

	t.Run("Success - User with no sequences", func(t *testing.T) {
		user3 := createTestUser(t, client, "user3@test.com", "User 3")

		sequences, err := service.ListSequences(ctx, user3.ID)

		require.NoError(t, err)
		assert.Len(t, sequences, 0)
	})
}

func TestUpdateSequence(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")

	sequence, err := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "Original Name",
		Trigger: "manual",
	})
	require.NoError(t, err)

	t.Run("Success - Update sequence", func(t *testing.T) {
		newName := "Updated Name"
		newDesc := "Updated Description"
		newStatus := "active"

		req := UpdateSequenceRequest{
			Name:        &newName,
			Description: &newDesc,
			Status:      &newStatus,
		}

		result, err := service.UpdateSequence(ctx, user.ID, sequence.ID, req)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", result.Name)
		assert.Equal(t, "Updated Description", result.Description)
		assert.Equal(t, "active", result.Status)
	})

	t.Run("Error - Sequence not found or unauthorized", func(t *testing.T) {
		user2 := createTestUser(t, client, "user2@test.com", "User 2")

		newName := "Hacked"
		req := UpdateSequenceRequest{Name: &newName}

		result, err := service.UpdateSequence(ctx, user2.ID, sequence.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found or unauthorized")
	})
}

func TestDeleteSequence(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")

	sequence, err := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "To Delete",
		Trigger: "manual",
	})
	require.NoError(t, err)

	t.Run("Success - Delete sequence", func(t *testing.T) {
		err := service.DeleteSequence(ctx, user.ID, sequence.ID)

		require.NoError(t, err)

		// Verify deleted
		_, err = service.GetSequence(ctx, sequence.ID)
		assert.Error(t, err)
	})

	t.Run("Error - Sequence not found or unauthorized", func(t *testing.T) {
		user2 := createTestUser(t, client, "user2@test.com", "User 2")

		err := service.DeleteSequence(ctx, user2.ID, 99999)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found or unauthorized")
	})
}

func TestCreateStep(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")

	sequence, err := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "Test Sequence",
		Trigger: "manual",
	})
	require.NoError(t, err)

	t.Run("Success - Create step", func(t *testing.T) {
		req := CreateStepRequest{
			SequenceID: sequence.ID,
			StepOrder:  1,
			DelayDays:  0,
			Subject:    "Welcome to IndustryDB",
			Body:       "Hi {{name}}, welcome to our platform!",
		}

		result, err := service.CreateStep(ctx, user.ID, req)

		require.NoError(t, err)
		assert.Equal(t, sequence.ID, result.SequenceID)
		assert.Equal(t, 1, result.StepOrder)
		assert.Equal(t, 0, result.DelayDays)
		assert.Equal(t, "Welcome to IndustryDB", result.Subject)
	})

	t.Run("Error - Sequence not found or unauthorized", func(t *testing.T) {
		user2 := createTestUser(t, client, "user2@test.com", "User 2")

		req := CreateStepRequest{
			SequenceID: sequence.ID,
			StepOrder:  1,
			Subject:    "Test",
			Body:       "Test",
		}

		result, err := service.CreateStep(ctx, user2.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetStep(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")

	sequence, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "Test",
		Trigger: "manual",
	})

	step, _ := service.CreateStep(ctx, user.ID, CreateStepRequest{
		SequenceID: sequence.ID,
		StepOrder:  1,
		Subject:    "Test Email",
		Body:       "Body",
	})

	t.Run("Success - Get step", func(t *testing.T) {
		result, err := service.GetStep(ctx, step.ID)

		require.NoError(t, err)
		assert.Equal(t, step.ID, result.ID)
		assert.Equal(t, "Test Email", result.Subject)
	})

	t.Run("Error - Step not found", func(t *testing.T) {
		result, err := service.GetStep(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestEnrollLead(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")
	lead := createTestLead(t, client, "Test Lead", "lead@test.com")

	// Create active sequence
	sequence, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "Test Sequence",
		Trigger: "manual",
	})

	// Activate sequence
	status := "active"
	service.UpdateSequence(ctx, user.ID, sequence.ID, UpdateSequenceRequest{
		Status: &status,
	})

	t.Run("Success - Enroll lead", func(t *testing.T) {
		req := EnrollLeadRequest{
			SequenceID: sequence.ID,
			LeadID:     lead.ID,
		}

		result, err := service.EnrollLead(ctx, user.ID, req)

		require.NoError(t, err)
		assert.Equal(t, sequence.ID, result.SequenceID)
		assert.Equal(t, lead.ID, result.LeadID)
		assert.Equal(t, "active", result.Status)
		assert.Equal(t, 0, result.CurrentStep)
	})

	t.Run("Error - Lead already enrolled", func(t *testing.T) {
		req := EnrollLeadRequest{
			SequenceID: sequence.ID,
			LeadID:     lead.ID,
		}

		result, err := service.EnrollLead(ctx, user.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "already enrolled")
	})

	t.Run("Error - Sequence not active", func(t *testing.T) {
		// Create draft sequence
		draftSeq, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
			Name:    "Draft Sequence",
			Trigger: "manual",
		})

		lead2 := createTestLead(t, client, "Lead 2", "lead2@test.com")

		req := EnrollLeadRequest{
			SequenceID: draftSeq.ID,
			LeadID:     lead2.ID,
		}

		result, err := service.EnrollLead(ctx, user.ID, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not active")
	})
}

func TestGetEnrollment(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")
	lead := createTestLead(t, client, "Test Lead", "lead@test.com")

	// Create and activate sequence
	sequence, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "Test Sequence",
		Trigger: "manual",
	})
	status := "active"
	service.UpdateSequence(ctx, user.ID, sequence.ID, UpdateSequenceRequest{Status: &status})

	enrollment, _ := service.EnrollLead(ctx, user.ID, EnrollLeadRequest{
		SequenceID: sequence.ID,
		LeadID:     lead.ID,
	})

	t.Run("Success - Get enrollment", func(t *testing.T) {
		result, err := service.GetEnrollment(ctx, enrollment.ID)

		require.NoError(t, err)
		assert.Equal(t, enrollment.ID, result.ID)
		assert.Equal(t, sequence.ID, result.SequenceID)
		assert.Equal(t, lead.ID, result.LeadID)
	})

	t.Run("Error - Enrollment not found", func(t *testing.T) {
		result, err := service.GetEnrollment(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestListLeadEnrollments(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")
	lead := createTestLead(t, client, "Test Lead", "lead@test.com")

	// Create 2 active sequences
	seq1, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{Name: "Seq 1", Trigger: "manual"})
	seq2, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{Name: "Seq 2", Trigger: "manual"})

	status := "active"
	service.UpdateSequence(ctx, user.ID, seq1.ID, UpdateSequenceRequest{Status: &status})
	service.UpdateSequence(ctx, user.ID, seq2.ID, UpdateSequenceRequest{Status: &status})

	// Enroll lead in both
	service.EnrollLead(ctx, user.ID, EnrollLeadRequest{SequenceID: seq1.ID, LeadID: lead.ID})
	service.EnrollLead(ctx, user.ID, EnrollLeadRequest{SequenceID: seq2.ID, LeadID: lead.ID})

	t.Run("Success - List enrollments", func(t *testing.T) {
		enrollments, err := service.ListLeadEnrollments(ctx, lead.ID)

		require.NoError(t, err)
		assert.Len(t, enrollments, 2)
	})
}

func TestStopEnrollment(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "owner@test.com", "Owner")
	lead := createTestLead(t, client, "Test Lead", "lead@test.com")

	// Create and activate sequence
	sequence, _ := service.CreateSequence(ctx, user.ID, CreateSequenceRequest{
		Name:    "Test Sequence",
		Trigger: "manual",
	})
	status := "active"
	service.UpdateSequence(ctx, user.ID, sequence.ID, UpdateSequenceRequest{Status: &status})

	enrollment, _ := service.EnrollLead(ctx, user.ID, EnrollLeadRequest{
		SequenceID: sequence.ID,
		LeadID:     lead.ID,
	})

	t.Run("Success - Stop enrollment", func(t *testing.T) {
		err := service.StopEnrollment(ctx, enrollment.ID)

		require.NoError(t, err)

		// Verify stopped
		result, _ := service.GetEnrollment(ctx, enrollment.ID)
		assert.Equal(t, "stopped", result.Status)
	})

	t.Run("Error - Enrollment not found", func(t *testing.T) {
		err := service.StopEnrollment(ctx, 99999)

		assert.Error(t, err)
	})
}
