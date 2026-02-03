package leadnote

import (
	"context"
	"testing"

	"github.com/jordanlanch/industrydb/ent"
	"github.com/jordanlanch/industrydb/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*ent.Client, func()) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return client, func() { client.Close() }
}

func createTestUser(t *testing.T, client *ent.Client, email, name string) *ent.User {
	user, err := client.User.
		Create().
		SetEmail(email).
		SetPasswordHash("hashed_password").
		SetName(name).
		Save(context.Background())
	require.NoError(t, err)
	return user
}

func createTestLead(t *testing.T, client *ent.Client) *ent.Lead {
	lead, err := client.Lead.
		Create().
		SetName("Test Tattoo Studio").
		SetIndustry("tattoo").
		SetCountry("US").
		SetCity("New York").
		Save(context.Background())
	require.NoError(t, err)
	return lead
}

func TestCreateNote(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "test@example.com", "Test User")
	lead := createTestLead(t, client)

	t.Run("Success - Create note", func(t *testing.T) {
		req := CreateNoteRequest{
			LeadID:   lead.ID,
			Content:  "Great lead! Called and confirmed contact info.",
			IsPinned: false,
		}

		note, err := service.CreateNote(ctx, user.ID, req)

		require.NoError(t, err)
		assert.NotNil(t, note)
		assert.Equal(t, lead.ID, note.LeadID)
		assert.Equal(t, user.ID, note.UserID)
		assert.Equal(t, user.Name, note.UserName)
		assert.Equal(t, req.Content, note.Content)
		assert.False(t, note.IsPinned)
		assert.NotZero(t, note.ID)
		assert.NotZero(t, note.CreatedAt)
		assert.NotZero(t, note.UpdatedAt)
	})

	t.Run("Success - Create pinned note", func(t *testing.T) {
		req := CreateNoteRequest{
			LeadID:   lead.ID,
			Content:  "IMPORTANT: Decision maker is John Doe",
			IsPinned: true,
		}

		note, err := service.CreateNote(ctx, user.ID, req)

		require.NoError(t, err)
		assert.True(t, note.IsPinned)
	})

	t.Run("Error - Invalid lead ID", func(t *testing.T) {
		req := CreateNoteRequest{
			LeadID:  99999, // Non-existent lead
			Content: "Test note",
		}

		note, err := service.CreateNote(ctx, user.ID, req)

		assert.Error(t, err)
		assert.Nil(t, note)
	})

	t.Run("Error - Invalid user ID", func(t *testing.T) {
		req := CreateNoteRequest{
			LeadID:  lead.ID,
			Content: "Test note",
		}

		note, err := service.CreateNote(ctx, 99999, req) // Non-existent user

		assert.Error(t, err)
		assert.Nil(t, note)
	})
}

func TestGetNoteByID(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "test@example.com", "Test User")
	lead := createTestLead(t, client)

	// Create a note
	createdNote, err := service.CreateNote(ctx, user.ID, CreateNoteRequest{
		LeadID:  lead.ID,
		Content: "Test note content",
	})
	require.NoError(t, err)

	t.Run("Success - Get existing note", func(t *testing.T) {
		note, err := service.GetNoteByID(ctx, createdNote.ID)

		require.NoError(t, err)
		assert.NotNil(t, note)
		assert.Equal(t, createdNote.ID, note.ID)
		assert.Equal(t, createdNote.Content, note.Content)
		assert.Equal(t, user.Name, note.UserName)
	})

	t.Run("Error - Note not found", func(t *testing.T) {
		note, err := service.GetNoteByID(ctx, 99999)

		assert.Error(t, err)
		assert.Nil(t, note)
		assert.Contains(t, err.Error(), "note not found")
	})
}

func TestListNotesByLead(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@example.com", "User One")
	user2 := createTestUser(t, client, "user2@example.com", "User Two")
	lead := createTestLead(t, client)

	t.Run("Empty list - No notes", func(t *testing.T) {
		notes, err := service.ListNotesByLead(ctx, lead.ID)

		require.NoError(t, err)
		assert.Empty(t, notes)
	})

	t.Run("Success - Multiple notes ordered correctly", func(t *testing.T) {
		// Create regular note
		_, err := service.CreateNote(ctx, user1.ID, CreateNoteRequest{
			LeadID:   lead.ID,
			Content:  "First note",
			IsPinned: false,
		})
		require.NoError(t, err)

		// Create pinned note (should appear first)
		_, err = service.CreateNote(ctx, user2.ID, CreateNoteRequest{
			LeadID:   lead.ID,
			Content:  "Pinned note",
			IsPinned: true,
		})
		require.NoError(t, err)

		// Create another regular note
		_, err = service.CreateNote(ctx, user1.ID, CreateNoteRequest{
			LeadID:   lead.ID,
			Content:  "Second note",
			IsPinned: false,
		})
		require.NoError(t, err)

		notes, err := service.ListNotesByLead(ctx, lead.ID)

		require.NoError(t, err)
		assert.Len(t, notes, 3)
		// First note should be pinned
		assert.True(t, notes[0].IsPinned)
		assert.Equal(t, "Pinned note", notes[0].Content)
		// Others should be unpinned and ordered by date (most recent first)
		assert.False(t, notes[1].IsPinned)
		assert.False(t, notes[2].IsPinned)
	})

	t.Run("Success - Different leads have separate notes", func(t *testing.T) {
		lead2 := createTestLead(t, client)

		_, err := service.CreateNote(ctx, user1.ID, CreateNoteRequest{
			LeadID:  lead2.ID,
			Content: "Note for lead 2",
		})
		require.NoError(t, err)

		notes1, err := service.ListNotesByLead(ctx, lead.ID)
		require.NoError(t, err)

		notes2, err := service.ListNotesByLead(ctx, lead2.ID)
		require.NoError(t, err)

		// lead should have 3 notes from previous test
		assert.Len(t, notes1, 3)
		// lead2 should have only 1 note
		assert.Len(t, notes2, 1)
	})
}

func TestUpdateNote(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@example.com", "User One")
	user2 := createTestUser(t, client, "user2@example.com", "User Two")
	lead := createTestLead(t, client)

	// Create a note
	createdNote, err := service.CreateNote(ctx, user1.ID, CreateNoteRequest{
		LeadID:  lead.ID,
		Content: "Original content",
	})
	require.NoError(t, err)

	t.Run("Success - Update content", func(t *testing.T) {
		newContent := "Updated content"
		req := UpdateNoteRequest{
			Content: &newContent,
		}

		note, err := service.UpdateNote(ctx, user1.ID, createdNote.ID, req)

		require.NoError(t, err)
		assert.Equal(t, newContent, note.Content)
		assert.Equal(t, createdNote.IsPinned, note.IsPinned)
	})

	t.Run("Success - Pin note", func(t *testing.T) {
		pinned := true
		req := UpdateNoteRequest{
			IsPinned: &pinned,
		}

		note, err := service.UpdateNote(ctx, user1.ID, createdNote.ID, req)

		require.NoError(t, err)
		assert.True(t, note.IsPinned)
	})

	t.Run("Success - Update both content and pinned", func(t *testing.T) {
		newContent := "Final content"
		unpinned := false
		req := UpdateNoteRequest{
			Content:  &newContent,
			IsPinned: &unpinned,
		}

		note, err := service.UpdateNote(ctx, user1.ID, createdNote.ID, req)

		require.NoError(t, err)
		assert.Equal(t, newContent, note.Content)
		assert.False(t, note.IsPinned)
	})

	t.Run("Error - Unauthorized (different user)", func(t *testing.T) {
		newContent := "Hacked content"
		req := UpdateNoteRequest{
			Content: &newContent,
		}

		note, err := service.UpdateNote(ctx, user2.ID, createdNote.ID, req)

		assert.Error(t, err)
		assert.Nil(t, note)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Error - Note not found", func(t *testing.T) {
		newContent := "New content"
		req := UpdateNoteRequest{
			Content: &newContent,
		}

		note, err := service.UpdateNote(ctx, user1.ID, 99999, req)

		assert.Error(t, err)
		assert.Nil(t, note)
		assert.Contains(t, err.Error(), "note not found")
	})
}

func TestDeleteNote(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user1 := createTestUser(t, client, "user1@example.com", "User One")
	user2 := createTestUser(t, client, "user2@example.com", "User Two")
	lead := createTestLead(t, client)

	t.Run("Success - Delete own note", func(t *testing.T) {
		note, err := service.CreateNote(ctx, user1.ID, CreateNoteRequest{
			LeadID:  lead.ID,
			Content: "Note to delete",
		})
		require.NoError(t, err)

		err = service.DeleteNote(ctx, user1.ID, note.ID)

		require.NoError(t, err)

		// Verify note is deleted
		deletedNote, err := service.GetNoteByID(ctx, note.ID)
		assert.Error(t, err)
		assert.Nil(t, deletedNote)
	})

	t.Run("Error - Unauthorized (different user)", func(t *testing.T) {
		note, err := service.CreateNote(ctx, user1.ID, CreateNoteRequest{
			LeadID:  lead.ID,
			Content: "Protected note",
		})
		require.NoError(t, err)

		err = service.DeleteNote(ctx, user2.ID, note.ID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		// Verify note still exists
		existingNote, err := service.GetNoteByID(ctx, note.ID)
		require.NoError(t, err)
		assert.NotNil(t, existingNote)
	})

	t.Run("Error - Note not found", func(t *testing.T) {
		err := service.DeleteNote(ctx, user1.ID, 99999)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "note not found")
	})
}

func TestCountNotesByLead(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	service := NewService(client)

	user := createTestUser(t, client, "test@example.com", "Test User")
	lead := createTestLead(t, client)

	t.Run("Zero count - No notes", func(t *testing.T) {
		count, err := service.CountNotesByLead(ctx, lead.ID)

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Success - Count multiple notes", func(t *testing.T) {
		// Create 3 notes
		for i := 0; i < 3; i++ {
			_, err := service.CreateNote(ctx, user.ID, CreateNoteRequest{
				LeadID:  lead.ID,
				Content: "Note",
			})
			require.NoError(t, err)
		}

		count, err := service.CountNotesByLead(ctx, lead.ID)

		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("Success - Count after deletion", func(t *testing.T) {
		notes, err := service.ListNotesByLead(ctx, lead.ID)
		require.NoError(t, err)

		// Delete one note
		err = service.DeleteNote(ctx, user.ID, notes[0].ID)
		require.NoError(t, err)

		count, err := service.CountNotesByLead(ctx, lead.ID)

		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}
