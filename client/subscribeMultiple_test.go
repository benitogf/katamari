package client_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/benitogf/katamari"
	"github.com/benitogf/katamari/client"
	"github.com/benitogf/katamari/key"
	"github.com/stretchr/testify/require"
)

// sleepAfterWrite sleeps only on Windows to allow file system operations to complete
func sleepAfterWrite() {
	if runtime.GOOS == "windows" {
		time.Sleep(10 * time.Millisecond)
	}
}

// Test types
type TestUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TestPost struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type TestComment struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

type TestLike struct {
	ID     int `json:"id"`
	UserID int `json:"user_id"`
}

// Helper functions to create test data
func createUser(t *testing.T, server *katamari.Server, id int, name string) {
	user := TestUser{ID: id, Name: name}
	data, err := json.Marshal(user)
	require.NoError(t, err)

	encoded := base64.StdEncoding.EncodeToString(data)
	server.Storage.Set(key.Build("users/*"), encoded)
}

func createPost(t *testing.T, server *katamari.Server, id int, title string) {
	post := TestPost{ID: id, Title: title}
	data, err := json.Marshal(post)
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(data)
	server.Storage.Set(key.Build("posts/*"), encoded)
}

func createComment(t *testing.T, server *katamari.Server, id int, text string) {
	comment := TestComment{ID: id, Text: text}
	data, err := json.Marshal(comment)
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(data)
	server.Storage.Set(key.Build("comments/*"), encoded)
}

func createLike(t *testing.T, server *katamari.Server, id int, userID int) {
	like := TestLike{ID: id, UserID: userID}
	data, err := json.Marshal(like)
	require.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(data)
	server.Storage.Set(key.Build("likes/*"), encoded)
}

func TestSubscribeMultiple2_BasicFunctionality(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var lastUsers []client.Meta[TestUser]
	var lastPosts []client.Meta[TestPost]

	wg := sync.WaitGroup{}
	// Wait for 2 initial callbacks (one per subscription path)
	wg.Add(2)

	go func() {
		client.SubscribeMultiple2(
			ctx,
			client.Path{Protocol: "ws", Host: server.Address, Path: "users/*"},
			client.Path{Protocol: "ws", Host: server.Address, Path: "posts/*"},
			func(users []client.Meta[TestUser], posts []client.Meta[TestPost]) {
				lastUsers = users
				lastPosts = posts
				wg.Done()
			},
		)
	}()

	// Wait for initial callbacks from both paths
	wg.Wait()

	// Create user and wait for callback
	wg.Add(1)
	createUser(t, &server, 1, "Alice")
	sleepAfterWrite()
	wg.Wait()

	// Create post and wait for callback
	wg.Add(1)
	createPost(t, &server, 1, "Test Post")
	sleepAfterWrite()
	wg.Wait()

	// Verify we received exactly 1 user
	require.Len(t, lastUsers, 1, "Expected exactly 1 user")
	require.Equal(t, 1, lastUsers[0].Data.ID, "Expected user ID to be 1")
	require.Equal(t, "Alice", lastUsers[0].Data.Name, "Expected user name to be 'Alice'")

	// Verify we received exactly 1 post
	require.Len(t, lastPosts, 1, "Expected exactly 1 post")
	require.Equal(t, 1, lastPosts[0].Data.ID, "Expected post ID to be 1")
	require.Equal(t, "Test Post", lastPosts[0].Data.Title, "Expected post title to be 'Test Post'")
}

func TestSubscribeMultiple2_StateAggregation(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var receivedStates []struct {
		userCount int
		postCount int
	}

	wg := sync.WaitGroup{}
	// Wait for 2 initial callbacks (one per subscription path)
	wg.Add(2)

	go func() {
		client.SubscribeMultiple2(
			ctx,
			client.Path{Protocol: "ws", Host: server.Address, Path: "users/*"},
			client.Path{Protocol: "ws", Host: server.Address, Path: "posts/*"},
			func(users []client.Meta[TestUser], posts []client.Meta[TestPost]) {
				receivedStates = append(receivedStates, struct {
					userCount int
					postCount int
				}{len(users), len(posts)})
				wg.Done()
			},
		)
	}()

	// Wait for initial callbacks from both paths
	wg.Wait()

	// Create multiple items, waiting after each write
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		createUser(t, &server, i, "User")
		sleepAfterWrite()
		wg.Wait()
	}
	for i := 1; i <= 2; i++ {
		wg.Add(1)
		createPost(t, &server, i, "Post")
		sleepAfterWrite()
		wg.Wait()
	}

	// Should receive: 2 initial + 5 writes (3 users + 2 posts) = 7 total
	require.Len(t, receivedStates, 7, "Expected exactly 7 state updates")

	// Verify initial state is empty
	require.Equal(t, 0, receivedStates[0].userCount, "Initial state should have 0 users")
	require.Equal(t, 0, receivedStates[0].postCount, "Initial state should have 0 posts")

	// Verify final state has all items
	finalState := receivedStates[len(receivedStates)-1]
	require.Equal(t, 3, finalState.userCount, "Final state should have 3 users")
	require.Equal(t, 2, finalState.postCount, "Final state should have 2 posts")
}

func TestSubscribeMultiple2_ContextCancellation(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	var callbackCount int

	wg := sync.WaitGroup{}
	// Wait for 2 initial callbacks (one per subscription path)
	wg.Add(2)

	go client.SubscribeMultiple2(
		ctx,
		client.Path{Protocol: "ws", Host: server.Address, Path: "users/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "posts/*"},
		func(users []client.Meta[TestUser], posts []client.Meta[TestPost]) {
			callbackCount++
			wg.Done()
		},
	)

	// Wait for initial callbacks from both paths
	wg.Wait()

	// Create some data before cancellation, waiting after each write
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		createUser(t, &server, i, "User")
		sleepAfterWrite()
		wg.Wait()
	}

	countBeforeCancel := callbackCount

	// Should have received: 2 initial + 3 writes = 5 callbacks
	require.Equal(t, 5, countBeforeCancel, "Expected exactly 5 callbacks before cancellation")

	// Cancel context
	cancel()
	time.Sleep(200 * time.Millisecond)

	// Create more data (should NOT be received after cancellation)
	for i := 4; i <= 6; i++ {
		createUser(t, &server, i, "User")
		sleepAfterWrite()
	}

	time.Sleep(200 * time.Millisecond)
	countAfterCancel := callbackCount

	// No new callbacks should be received after cancellation
	require.Equal(t, countBeforeCancel, countAfterCancel, "Expected no new callbacks after context cancellation")
}

func TestSubscribeMultiple3_BasicFunctionality(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var lastUsers []client.Meta[TestUser]
	var lastPosts []client.Meta[TestPost]
	var lastComments []client.Meta[TestComment]

	wg := sync.WaitGroup{}
	// Wait for 3 initial callbacks (one per subscription path)
	wg.Add(3)

	go client.SubscribeMultiple3(
		ctx,
		client.Path{Protocol: "ws", Host: server.Address, Path: "users/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "posts/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "comments/*"},
		func(users []client.Meta[TestUser], posts []client.Meta[TestPost], comments []client.Meta[TestComment]) {
			lastUsers = users
			lastPosts = posts
			lastComments = comments
			wg.Done()
		},
	)

	// Wait for initial callbacks from all paths
	wg.Wait()

	// Create user and wait for callback
	wg.Add(1)
	createUser(t, &server, 1, "Alice")
	sleepAfterWrite()
	wg.Wait()

	// Create post and wait for callback
	wg.Add(1)
	createPost(t, &server, 1, "Test Post")
	sleepAfterWrite()
	wg.Wait()

	// Create comment and wait for callback
	wg.Add(1)
	createComment(t, &server, 1, "Great post!")
	sleepAfterWrite()
	wg.Wait()

	// Verify exactly 1 user
	require.Len(t, lastUsers, 1, "Expected exactly 1 user")
	require.Equal(t, 1, lastUsers[0].Data.ID, "Expected user ID to be 1")
	require.Equal(t, "Alice", lastUsers[0].Data.Name, "Expected user name to be 'Alice'")

	// Verify exactly 1 post
	require.Len(t, lastPosts, 1, "Expected exactly 1 post")
	require.Equal(t, 1, lastPosts[0].Data.ID, "Expected post ID to be 1")
	require.Equal(t, "Test Post", lastPosts[0].Data.Title, "Expected post title to be 'Test Post'")

	// Verify exactly 1 comment
	require.Len(t, lastComments, 1, "Expected exactly 1 comment")
	require.Equal(t, 1, lastComments[0].Data.ID, "Expected comment ID to be 1")
	require.Equal(t, "Great post!", lastComments[0].Data.Text, "Expected comment text to be 'Great post!'")
}

func TestSubscribeMultiple4_BasicFunctionality(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var lastUsers []client.Meta[TestUser]
	var lastPosts []client.Meta[TestPost]
	var lastComments []client.Meta[TestComment]
	var lastLikes []client.Meta[TestLike]

	wg := sync.WaitGroup{}
	// Wait for 4 initial callbacks (one per subscription path)
	wg.Add(4)

	go client.SubscribeMultiple4(
		ctx,
		client.Path{Protocol: "ws", Host: server.Address, Path: "users/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "posts/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "comments/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "likes/*"},
		func(users []client.Meta[TestUser], posts []client.Meta[TestPost], comments []client.Meta[TestComment], likes []client.Meta[TestLike]) {
			lastUsers = users
			lastPosts = posts
			lastComments = comments
			lastLikes = likes
			wg.Done()
		},
	)

	// Wait for initial callbacks from all paths
	wg.Wait()

	// Create user and wait for callback
	wg.Add(1)
	createUser(t, &server, 1, "Alice")
	sleepAfterWrite()
	wg.Wait()

	// Create post and wait for callback
	wg.Add(1)
	createPost(t, &server, 1, "Test Post")
	sleepAfterWrite()
	wg.Wait()

	// Create comment and wait for callback
	wg.Add(1)
	createComment(t, &server, 1, "Great!")
	sleepAfterWrite()
	wg.Wait()

	// Create like and wait for callback
	wg.Add(1)
	createLike(t, &server, 1, 1)
	sleepAfterWrite()
	wg.Wait()

	// Verify exactly 1 user
	require.Len(t, lastUsers, 1, "Expected exactly 1 user")
	require.Equal(t, 1, lastUsers[0].Data.ID, "Expected user ID to be 1")
	require.Equal(t, "Alice", lastUsers[0].Data.Name, "Expected user name to be 'Alice'")

	// Verify exactly 1 post
	require.Len(t, lastPosts, 1, "Expected exactly 1 post")
	require.Equal(t, 1, lastPosts[0].Data.ID, "Expected post ID to be 1")
	require.Equal(t, "Test Post", lastPosts[0].Data.Title, "Expected post title to be 'Test Post'")

	// Verify exactly 1 comment
	require.Len(t, lastComments, 1, "Expected exactly 1 comment")
	require.Equal(t, 1, lastComments[0].Data.ID, "Expected comment ID to be 1")
	require.Equal(t, "Great!", lastComments[0].Data.Text, "Expected comment text to be 'Great!'")

	// Verify exactly 1 like
	require.Len(t, lastLikes, 1, "Expected exactly 1 like")
	require.Equal(t, 1, lastLikes[0].Data.ID, "Expected like ID to be 1")
	require.Equal(t, 1, lastLikes[0].Data.UserID, "Expected like user ID to be 1")
}

func TestSubscribeMultiple2_ConcurrentUpdates(t *testing.T) {
	server := katamari.Server{}
	server.Silence = true
	server.Start("localhost:0")
	defer server.Close(os.Interrupt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var allStates []struct {
		users []client.Meta[TestUser]
		posts []client.Meta[TestPost]
	}

	wg := sync.WaitGroup{}
	// Wait for 2 initial callbacks (one per subscription path)
	wg.Add(2)

	go client.SubscribeMultiple2(
		ctx,
		client.Path{Protocol: "ws", Host: server.Address, Path: "users/*"},
		client.Path{Protocol: "ws", Host: server.Address, Path: "posts/*"},
		func(users []client.Meta[TestUser], posts []client.Meta[TestPost]) {
			// Make copies to avoid race conditions
			usersCopy := make([]client.Meta[TestUser], len(users))
			postsCopy := make([]client.Meta[TestPost], len(posts))
			copy(usersCopy, users)
			copy(postsCopy, posts)
			allStates = append(allStates, struct {
				users []client.Meta[TestUser]
				posts []client.Meta[TestPost]
			}{usersCopy, postsCopy})
			wg.Done()
		},
	)

	// Wait for initial callbacks from both paths
	wg.Wait()

	// Create data rapidly, waiting after each write
	for i := range 5 {
		wg.Add(1)
		createUser(t, &server, i, "User")
		sleepAfterWrite()
	}
	for i := range 5 {
		wg.Add(1)
		createPost(t, &server, i, "Post")
		sleepAfterWrite()
	}

	wg.Wait()

	// Should receive: 2 initial + 10 writes (5 users + 5 posts) = 12 total
	require.Len(t, allStates, 12, "Expected exactly 12 state updates")

	// Verify initial state is empty
	require.Equal(t, 0, len(allStates[0].users), "Initial state should have 0 users")
	require.Equal(t, 0, len(allStates[0].posts), "Initial state should have 0 posts")

	// Verify final state has all items
	finalState := allStates[len(allStates)-1]
	require.Len(t, finalState.users, 5, "Final state should have exactly 5 users")
	require.Len(t, finalState.posts, 5, "Final state should have exactly 5 posts")
}

func TestPath_Struct(t *testing.T) {
	// Test that Path struct works correctly
	path := client.Path{
		Protocol: "ws",
		Host:     "localhost:8080",
		Path:     "/data/users/*",
	}

	require.Equal(t, "ws", path.Protocol)
	require.Equal(t, "localhost:8080", path.Host)
	require.Equal(t, "/data/users/*", path.Path)
}
