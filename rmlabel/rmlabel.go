package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// This is some very hacky code to bulk remove labels from emails.
//
// Once you have a label with enough messages attached, it becomes impossible to
// directly delete the label. Removing the label from all the emails in one big
// go also fails. The only solution I've found is to gradually remove the label
// from the emails in small batches. This code automates that.
//
// Some code is copied from https://developers.google.com/gmail/api/quickstart/go.
//
// os.Args[1] is a path to credentials.json. This needs the gmail.modify
// permission. See gmailctl (for example) for instructions on how to set up a
// gcloud project and associated app and oauth client ID.
//
// os.Args[2] is the name of the label to remove all emails for.
//
// TODO: clean this up. If I need it again, turn it into a nicer tool for bulk
// adding/removing labels.

func main() {
	log.SetFlags(0)
	ctx := context.Background()
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	conf, err := google.ConfigFromJSON(b, gmail.GmailModifyScope)
	if err != nil {
		log.Fatal(err)
	}
	client := getClient(conf)

	svc, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatal(err)
	}

	r, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		log.Fatal(err)
	}
	labelName := os.Args[2]
	var labelID string
	for _, label := range r.Labels {
		if label.Name == labelName {
			labelID = label.Id
			break
		}
	}
	if labelID == "" {
		log.Fatalf("No label named %q", labelName)
	}
	log.Printf("Dissociating messages for label %q (%s)...", labelName, labelID)

	var n int64
	for {
		ids, err := listNextBatch(svc, labelID)
		if err != nil {
			log.Fatalln("Error listing batch:", err)
		}
		if len(ids) == 0 {
			break
		}
		n += int64(len(ids))
		log.Printf(
			"Handled %d messages; got next batch of %d messages",
			n, len(ids),
		)
		if err := removeLabels(svc, ids, labelID); err != nil {
			log.Fatalln("Error removing label from batch:", err)
		}
	}
	log.Printf("Removed label %q from %d messages", labelName, n)
}

func listNextBatch(svc *gmail.Service, labelID string) ([]string, error) {
	req := svc.Users.Messages.List("me")
	req.MaxResults(500)
	req.LabelIds(labelID)
	result, err := req.Do()
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(result.Messages))
	for i, m := range result.Messages {
		ids[i] = m.Id
	}
	return ids, nil
}

func removeLabels(svc *gmail.Service, ids []string, labelID string) error {
	req := &gmail.BatchModifyMessagesRequest{
		Ids:            ids,
		RemoveLabelIds: []string{labelID},
	}
	return svc.Users.Messages.BatchModify("me", req).Do()
}

func getClient(conf *oauth2.Config) *http.Client {
	const name = "token.json"
	tok, err := tokenFromFile(name)
	if err != nil {
		tok = getTokenFromWeb(conf)
		saveToken(name, tok)
	}
	return conf.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
