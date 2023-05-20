package api

import (
	"log"
	"net/http"

	"github.com/nicklaw5/helix/v2"
	"github.com/saxypandabear/twitchsongrequests/internal/util"
	"github.com/saxypandabear/twitchsongrequests/pkg/db"
)

const (
	topicVersion = "1"
	subMethod    = "webhook"
)

type SubscribeRequest struct {
	UserID string `json:"user_id"`
}

type EventSubHandler struct {
	auth        *util.AuthConfig
	userStore   db.UserStore
	callbackURL string
	secret      string
}

func NewEventSubHandler(u db.UserStore, auth *util.AuthConfig, callbackURL, secret string) *EventSubHandler {
	return &EventSubHandler{
		userStore:   u,
		auth:        auth,
		callbackURL: callbackURL,
		secret:      secret,
	}
}

// SubscribeToTopic
func (e *EventSubHandler) SubscribeToTopic(w http.ResponseWriter, r *http.Request) {
	id, err := util.GetUserIDFromRequest(r)
	if err != nil {
		log.Println("failed to get Twitch ID from request", err)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}

	user, err := e.userStore.GetUser(id)
	if err != nil {
		log.Println("failed to get user", err)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}

	// TODO: figure this out
	// createReward := helix.ChannelCustomRewardsParams{
	// 	Title: "Spotify Song Request",
	// 	IsUserInputRequired: true,
	// 	IsEnabled: true,
	// 	Cost: 1000,
	// 	Prompt: "Request with a Spotify URL",
	// }

	createSub := helix.EventSubSubscription{
		Type:    helix.EventSubTypeChannelPointsCustomRewardRedemptionAdd,
		Version: topicVersion,
		Condition: helix.EventSubCondition{
			BroadcasterUserID: id,
		},
		Transport: helix.EventSubTransport{
			Method:   subMethod,
			Callback: e.callbackURL + "/callback",
			Secret:   e.secret,
		},
	}

	// get user access token
	c, err := util.GetNewTwitchClient(e.auth)
	if err != nil {
		log.Println("failed to get Twitch client for", id, err)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}

	token, err := c.RequestAppAccessToken([]string{e.auth.Scope})
	if err != nil {
		log.Println("failed to get updated access token for", id, err)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}
	c.SetAppAccessToken(token.Data.AccessToken)

	res, err := c.CreateEventSubSubscription(&createSub)
	if err != nil {
		log.Println("failed to create EventSub subscription ", err)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	} else if len(res.ErrorMessage) > 0 {
		log.Printf("error occurred while creating EventSub subscription | HTTP %v | %s | %s\n", res.ErrorStatus, res.Error, res.ErrorMessage)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}

	log.Println("Subscriptions:", res.Data.EventSubSubscriptions)

	if len(res.Data.EventSubSubscriptions) < 1 {
		log.Println("failed to subscribe to webhook event")
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}

	// successfully subscribed
	user.Subscribed = true
	user.SubscriptionID = res.Data.EventSubSubscriptions[0].ID
	err = e.userStore.UpdateUser(user)
	if err != nil {
		log.Println("failed to update twitch credentials", err)
		http.Redirect(w, r, e.callbackURL, http.StatusFound)
		return
	}

	log.Println("successfully subscribed to Channel Point topic for user", id)

	http.Redirect(w, r, e.callbackURL, http.StatusFound)
}
