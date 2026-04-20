package core

import (
	"sync"
	"testing"
)

func TestAgentMessageBus_CreateAndDeleteTeam(t *testing.T) {
	bus := NewAgentMessageBus()

	team, err := bus.CreateTeam("test-team", "A test team")
	if err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if team.ID == "" {
		t.Fatal("team ID should not be empty")
	}
	if team.Name != "test-team" {
		t.Fatalf("expected name 'test-team', got %q", team.Name)
	}

	// Get team
	got, err := bus.GetTeam(team.ID)
	if err != nil {
		t.Fatalf("GetTeam failed: %v", err)
	}
	if got.ID != team.ID {
		t.Fatalf("expected ID %q, got %q", team.ID, got.ID)
	}

	// List teams
	teams := bus.ListTeams()
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}

	// Delete team
	if err := bus.DeleteTeam(team.ID); err != nil {
		t.Fatalf("DeleteTeam failed: %v", err)
	}

	// Should not exist anymore
	_, err = bus.GetTeam(team.ID)
	if err == nil {
		t.Fatal("expected error after deleting team")
	}
}

func TestAgentMessageBus_JoinAndLeave(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")

	// Join team
	if err := bus.JoinTeam(team.ID, "agent-a", "task_1"); err != nil {
		t.Fatalf("JoinTeam failed: %v", err)
	}

	// Verify membership
	team, _ = bus.GetTeam(team.ID)
	if len(team.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(team.Members))
	}
	member, ok := team.Members["agent-a"]
	if !ok {
		t.Fatal("agent-a should be a member")
	}
	if member.TaskID != "task_1" {
		t.Fatalf("expected task_id 'task_1', got %q", member.TaskID)
	}

	// Join another agent
	bus.JoinTeam(team.ID, "agent-b", "task_2")
	team, _ = bus.GetTeam(team.ID)
	if len(team.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(team.Members))
	}

	// Leave team
	if err := bus.LeaveTeam(team.ID, "agent-a"); err != nil {
		t.Fatalf("LeaveTeam failed: %v", err)
	}
	team, _ = bus.GetTeam(team.ID)
	if len(team.Members) != 1 {
		t.Fatalf("expected 1 member after leave, got %d", len(team.Members))
	}
}

func TestAgentMessageBus_DirectMessage(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "alice", "task_1")
	bus.JoinTeam(team.ID, "bob", "task_2")

	// Alice sends message to Bob
	msg, err := bus.SendMessage(team.ID, "alice", "bob", MessageDirect, "Hello Bob!", "Greeting from Alice")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg.From != "alice" || msg.To != "bob" {
		t.Fatalf("unexpected message: from=%q to=%q", msg.From, msg.To)
	}

	// Bob receives the message
	messages := bus.ReceiveMessages("bob")
	if len(messages) != 1 {
		t.Fatalf("expected 1 message for bob, got %d", len(messages))
	}
	if messages[0].Content != "Hello Bob!" {
		t.Fatalf("unexpected content: %q", messages[0].Content)
	}

	// Alice's mailbox should be empty
	aliceMsgs := bus.ReceiveMessages("alice")
	if len(aliceMsgs) != 0 {
		t.Fatalf("expected 0 messages for alice, got %d", len(aliceMsgs))
	}
}

func TestAgentMessageBus_Broadcast(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "alice", "task_1")
	bus.JoinTeam(team.ID, "bob", "task_2")
	bus.JoinTeam(team.ID, "charlie", "task_3")

	// Alice broadcasts
	_, err := bus.SendMessage(team.ID, "alice", "", MessageBroadcast, "Hello everyone!", "Team announcement")
	if err != nil {
		t.Fatalf("Broadcast failed: %v", err)
	}

	// Bob and Charlie should receive, Alice should not
	for _, agent := range []string{"bob", "charlie"} {
		msgs := bus.ReceiveMessages(agent)
		if len(msgs) != 1 {
			t.Fatalf("expected 1 broadcast for %s, got %d", agent, len(msgs))
		}
	}
	aliceMsgs := bus.ReceiveMessages("alice")
	if len(aliceMsgs) != 0 {
		t.Fatalf("expected 0 broadcasts for sender, got %d", len(aliceMsgs))
	}
}

func TestAgentMessageBus_UpdateMemberStatus(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "worker", "task_1")

	// Update status
	if err := bus.UpdateMemberStatus(team.ID, "worker", "completed", "Done!"); err != nil {
		t.Fatalf("UpdateMemberStatus failed: %v", err)
	}

	team, _ = bus.GetTeam(team.ID)
	if team.Members["worker"].Status != "completed" {
		t.Fatalf("expected status 'completed', got %q", team.Members["worker"].Status)
	}
	if team.Members["worker"].Result != "Done!" {
		t.Fatalf("expected result 'Done!', got %q", team.Members["worker"].Result)
	}
}

func TestAgentMessageBus_ConcurrentAccess(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "receiver", "task_1")

	// Concurrent message sending
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = bus.SendMessage(team.ID, "sender", "receiver", MessageDirect,
				"message from sender", "test message")
		}()
	}
	wg.Wait()

	// Should receive at least some messages (mailbox has capacity 64)
	msgs := bus.ReceiveMessages("receiver")
	if len(msgs) == 0 {
		t.Fatal("expected some messages after concurrent send")
	}
}

func TestAgentMessageBus_WaitMailbox(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "agent", "task_1")

	// Get mailbox channel
	mailbox, err := bus.WaitMailbox("agent")
	if err != nil {
		t.Fatalf("WaitMailbox failed: %v", err)
	}

	// Send a message
	go func() {
		_, _ = bus.SendMessage(team.ID, "other", "agent", MessageDirect, "async msg", "test")
	}()

	// Should receive via channel
	msg := <-mailbox
	if msg.Content != "async msg" {
		t.Fatalf("unexpected content: %q", msg.Content)
	}
}

func TestAgentMessageBus_ShutdownRequest(t *testing.T) {
	bus := NewAgentMessageBus()
	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "worker", "task_1")

	// Send shutdown request
	msg, err := bus.SendMessage(team.ID, "main", "worker", MessageShutdownRequest,
		"Work is done, please shut down", "Shutdown request")
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if msg.Type != MessageShutdownRequest {
		t.Fatalf("expected type %q, got %q", MessageShutdownRequest, msg.Type)
	}

	// Worker receives shutdown request
	msgs := bus.ReceiveMessages("worker")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Type != MessageShutdownRequest {
		t.Fatalf("expected shutdown_request, got %q", msgs[0].Type)
	}
}

func TestAgentMessageBus_TeamMemberCount(t *testing.T) {
	bus := NewAgentMessageBus()

	// Non-existent team
	count := bus.TeamMemberCount("nonexistent")
	if count != 0 {
		t.Fatalf("expected 0 for nonexistent team, got %d", count)
	}

	team, _ := bus.CreateTeam("test", "desc")
	bus.JoinTeam(team.ID, "a", "t1")
	bus.JoinTeam(team.ID, "b", "t2")

	count = bus.TeamMemberCount(team.ID)
	if count != 2 {
		t.Fatalf("expected 2 members, got %d", count)
	}
}
