package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trungquantrannguyen/threadly/db"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const seedNamespace = "threadly-seed-v1"

type seedPlan struct {
	Users         []dbmodel.User
	Posts         []dbmodel.Post
	UserPostCount map[uuid.UUID]int
}

func main() {
	var (
		reset    = flag.Bool("reset", false, "delete existing seed users/posts before inserting")
		users    = flag.Int("users", 50, "number of seed users to create")
		minPosts = flag.Int("min-posts", 3, "minimum root posts per user")
		maxPosts = flag.Int("max-posts", 4, "maximum root posts per user")
	)

	flag.Parse()

	if *users <= 0 {
		panic("users must be greater than 0")
	}

	if *minPosts <= 0 || *maxPosts < *minPosts {
		panic("invalid post range")
	}

	seedPassword := os.Getenv("SEED_USER_PASSWORD")
	if seedPassword == "" {
		seedPassword = "Password123!"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Errorf("failed to hash seed password: %w", err))
	}

	cfg := config.Load("seed", "0")

	dtb, err := db.ConnectPostgres(cfg)
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}

	plan := buildSeedPlan(*users, *minPosts, *maxPosts, string(passwordHash))

	if err := runSeed(context.Background(), dtb, plan, *reset); err != nil {
		panic(err)
	}

	rootPostCount := 0
	replyPostCount := 0

	for _, post := range plan.Posts {
		if post.ReplyToPostID == nil {
			rootPostCount++
		} else {
			replyPostCount++
		}
	}

	fmt.Println("Seed completed successfully")
	fmt.Printf("Users: %d\n", len(plan.Users))
	fmt.Printf("Root posts: %d\n", rootPostCount)
	fmt.Printf("Reply/nested reply posts: %d\n", replyPostCount)
	fmt.Printf("Default seed password: %s\n", seedPassword)
}

func runSeed(ctx context.Context, dtb *gorm.DB, plan seedPlan, reset bool) error {
	return dtb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if reset {
			if err := resetSeedData(tx); err != nil {
				return err
			}
		}

		if err := upsertUsers(tx, plan.Users); err != nil {
			return fmt.Errorf("failed to seed users: %w", err)
		}

		if err := upsertPosts(tx, plan.Posts); err != nil {
			return fmt.Errorf("failed to seed posts: %w", err)
		}

		return nil
	})
}

func resetSeedData(tx *gorm.DB) error {
	var seedUserIDs []uuid.UUID

	if err := tx.
		Unscoped().
		Model(&dbmodel.User{}).
		Where("username LIKE ?", "seed_user_%").
		Pluck("id", &seedUserIDs).Error; err != nil {
		return fmt.Errorf("failed to find old seed users: %w", err)
	}

	if len(seedUserIDs) > 0 {
		if err := tx.
			Unscoped().
			Where("author_id IN ?", seedUserIDs).
			Delete(&dbmodel.Post{}).Error; err != nil {
			return fmt.Errorf("failed to delete old seed posts: %w", err)
		}
	}

	if err := tx.
		Unscoped().
		Where("username LIKE ?", "seed_user_%").
		Delete(&dbmodel.User{}).Error; err != nil {
		return fmt.Errorf("failed to delete old seed users: %w", err)
	}

	return nil
}

func upsertUsers(tx *gorm.DB, users []dbmodel.User) error {
	if len(users) == 0 {
		return nil
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"email",
			"username",
			"password_hash",
			"display_name",
			"bio",
			"avatar_url",
			"banner_url",
			"location",
			"website_url",
			"is_verified",
			"follower_count",
			"following_count",
			"post_count",
			"role",
			"updated_at",
			"deleted_at",
		}),
	}).Create(&users).Error
}

func upsertPosts(tx *gorm.DB, posts []dbmodel.Post) error {
	if len(posts) == 0 {
		return nil
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"author_id",
			"reply_to_post_id",
			"content",
			"visibility",
			"like_count",
			"reply_count",
			"repost_count",
			"bookmark_count",
			"created_at",
			"updated_at",
			"deleted_at",
		}),
	}).CreateInBatches(posts, 250).Error
}

func buildSeedPlan(userCount int, minPosts int, maxPosts int, passwordHash string) seedPlan {
	r := rand.New(rand.NewSource(2209))

	users := make([]dbmodel.User, 0, userCount)
	posts := make([]dbmodel.Post, 0)
	replyCounts := make(map[uuid.UUID]int)
	userPostCount := make(map[uuid.UUID]int)

	now := time.Now().UTC()

	for i := 1; i <= userCount; i++ {
		username := fmt.Sprintf("seed_user_%02d", i)
		email := fmt.Sprintf("%s@threadly.local", username)
		displayName := seedDisplayName(i)
		bio := seedBio(i)
		location := seedLocation(i)
		avatarURL := fmt.Sprintf("https://api.dicebear.com/9.x/initials/svg?seed=%s", username)
		websiteURL := fmt.Sprintf("https://threadly.local/%s", username)

		createdAt := now.Add(-time.Duration(userCount-i+10) * 12 * time.Hour)

		user := dbmodel.User{
			ID:           stableID("user", username),
			Email:        email,
			Username:     username,
			PasswordHash: passwordHash,
			DisplayName:  displayName,
			Bio:          &bio,
			AvatarURL:    &avatarURL,
			Location:     &location,
			WebsiteURL:   &websiteURL,
			IsVerified:   i%10 == 0,
			Role:         "user",
			CreatedAt:    createdAt,
			UpdatedAt:    createdAt,
		}

		users = append(users, user)
		userPostCount[user.ID] = 0
	}

	addPost := func(post dbmodel.Post) {
		posts = append(posts, post)
		userPostCount[post.AuthorID]++
	}

	rootPosts := make([]dbmodel.Post, 0)

	for userIndex, user := range users {
		rootPostTotal := minPosts + r.Intn(maxPosts-minPosts+1)

		for postIndex := 1; postIndex <= rootPostTotal; postIndex++ {
			postID := stableID("root-post", user.Username, fmt.Sprintf("%02d", postIndex))
			createdAt := now.
				Add(-14 * 24 * time.Hour).
				Add(time.Duration(len(rootPosts)) * 47 * time.Minute)

			post := dbmodel.Post{
				ID:            postID,
				AuthorID:      user.ID,
				Content:       seedRootPostContent(userIndex, postIndex),
				Visibility:    "public",
				LikeCount:     r.Intn(40),
				ReplyCount:    0,
				RepostCount:   r.Intn(8),
				BookmarkCount: r.Intn(15),
				CreatedAt:     createdAt,
				UpdatedAt:     createdAt,
			}

			rootPosts = append(rootPosts, post)
			addPost(post)
		}
	}

	for rootIndex, rootPost := range rootPosts {
		directReplyTotal := 4 + r.Intn(2)

		for replyIndex := 1; replyIndex <= directReplyTotal; replyIndex++ {
			replyAuthor := users[r.Intn(len(users))]
			parentID := rootPost.ID
			replyID := stableID(
				"reply",
				rootPost.ID.String(),
				fmt.Sprintf("%02d", replyIndex),
			)

			createdAt := rootPost.CreatedAt.
				Add(time.Duration(replyIndex) * 19 * time.Minute)

			reply := dbmodel.Post{
				ID:            replyID,
				AuthorID:      replyAuthor.ID,
				ReplyToPostID: &parentID,
				Content:       seedReplyContent(replyAuthor.Username, replyIndex),
				Visibility:    "public",
				LikeCount:     r.Intn(15),
				ReplyCount:    0,
				RepostCount:   r.Intn(3),
				BookmarkCount: r.Intn(5),
				CreatedAt:     createdAt,
				UpdatedAt:     createdAt,
			}

			replyCounts[rootPost.ID]++
			addPost(reply)

			shouldCreateNestedReply := (rootIndex+replyIndex)%3 == 0
			if shouldCreateNestedReply {
				nestedAuthor := users[r.Intn(len(users))]
				nestedParentID := reply.ID
				nestedReplyID := stableID(
					"nested-reply",
					reply.ID.String(),
					"01",
				)

				nestedCreatedAt := reply.CreatedAt.Add(11 * time.Minute)

				nestedReply := dbmodel.Post{
					ID:            nestedReplyID,
					AuthorID:      nestedAuthor.ID,
					ReplyToPostID: &nestedParentID,
					Content:       seedNestedReplyContent(nestedAuthor.Username),
					Visibility:    "public",
					LikeCount:     r.Intn(8),
					ReplyCount:    0,
					RepostCount:   r.Intn(2),
					BookmarkCount: r.Intn(3),
					CreatedAt:     nestedCreatedAt,
					UpdatedAt:     nestedCreatedAt,
				}

				replyCounts[reply.ID]++
				addPost(nestedReply)
			}
		}
	}

	for i := range posts {
		posts[i].ReplyCount = replyCounts[posts[i].ID]
	}

	for i := range users {
		users[i].PostCount = userPostCount[users[i].ID]
	}

	return seedPlan{
		Users:         users,
		Posts:         posts,
		UserPostCount: userPostCount,
	}
}

func stableID(parts ...string) uuid.UUID {
	key := seedNamespace + ":" + strings.Join(parts, ":")
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(key))
}

func seedDisplayName(index int) string {
	firstNames := []string{
		"Avery", "Mina", "Leo", "Nora", "Kai",
		"Sofia", "Ethan", "Linh", "Noah", "Maya",
		"Aria", "Theo", "Ivy", "Lucas", "Chloe",
		"Ryan", "Emma", "Hugo", "Zoe", "Owen",
	}

	lastNames := []string{
		"Nguyen", "Tran", "Pham", "Le", "Hoang",
		"Brown", "Smith", "Wilson", "Taylor", "Clark",
	}

	first := firstNames[(index-1)%len(firstNames)]
	last := lastNames[(index-1)%len(lastNames)]

	return fmt.Sprintf("%s %s", first, last)
}

func seedBio(index int) string {
	bios := []string{
		"Building small things and sharing what I learn.",
		"Backend learner, coffee drinker, and product thinker.",
		"Posting about code, design, and daily progress.",
		"Trying to make social apps feel simple again.",
		"Learning Go, PostgreSQL, Redis, and distributed systems.",
	}

	return bios[(index-1)%len(bios)]
}

func seedLocation(index int) string {
	locations := []string{
		"Melbourne",
		"Sydney",
		"Brisbane",
		"Canberra",
		"Ho Chi Minh City",
		"Da Nang",
		"Singapore",
		"Tokyo",
	}

	return locations[(index-1)%len(locations)]
}

func seedRootPostContent(userIndex int, postIndex int) string {
	templates := []string{
		"Working on Threadly today. The microservice structure is starting to feel much cleaner.",
		"Small backend win: one clear model, one repository layer, and fewer random duplicates.",
		"PostgreSQL plus GORM feels productive once the model tags are consistent.",
		"Thinking about how replies should behave when every reply is also just a post.",
		"Redis caching will make the home feed much faster once the basic content flow is stable.",
		"RabbitMQ makes more sense when notifications and feed updates become async events.",
		"Clean architecture is easier when the gateway owns REST and services own business logic.",
		"Today I am testing seed data so the frontend has realistic posts and replies to display.",
	}

	template := templates[(userIndex+postIndex)%len(templates)]
	return fmt.Sprintf("%s #%02d", template, postIndex)
}

func seedReplyContent(username string, replyIndex int) string {
	templates := []string{
		"Agree with this. The structure is easier to reason about now.",
		"This will be useful when the frontend starts rendering real timelines.",
		"Nice progress. The reply model is simple but flexible.",
		"Good point. Keeping replies in the posts table avoids extra complexity.",
		"I like this approach. It should make nested replies easier later.",
	}

	template := templates[(replyIndex-1)%len(templates)]
	return fmt.Sprintf("@%s %s", username, template)
}

func seedNestedReplyContent(username string) string {
	templates := []string{
		"Exactly. This is also a good test for nested reply rendering.",
		"That makes sense. The frontend can treat it as another post node.",
		"Yes, and the reply_count should only count direct children.",
		"Good example for testing conversation threads.",
	}

	index := int(stableID("nested-content", username)[0]) % len(templates)

	return fmt.Sprintf("@%s %s", username, templates[index])
}
