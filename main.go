package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

type Meeting struct {
	ID           primitive.ObjectID `json:"_id" bson:"_id"`
	Title        string             `json:"title" bson:"title"`
	Participants []Participant      `json:"participants" bson:"participant"`
	Start        time.Time          `json:"starttime" bson:"starttime"`
	End          time.Time          `json:"endtime" bson:"endtime"`
	Created      time.Time          `json:"createdtime" bson:"createdtime"`
}

type Participant struct {
	Name  string `json:"name" bson:"name"`
	Email string `json:"email" bson:"email"`
	RSVP  string `json:"rsvp" bson:"rsvp"`
}

func CreateParticipant(participant Participant, meeting Meeting) (error, string) {
	if participant.Name == "" || participant.Email == "" || participant.RSVP == "" {
		return errors.New("Fill All Details"), string(0)
	}

	var high Participant

	collection := client.Database("db").Collection("participant")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	err := collection.FindOne(ctx, bson.M{"email": participant.Email}).Decode(&high)

	var data Participant
	err = collection.FindOne(ctx, bson.M{"email": participant.Email}).Decode(&data)

	if !CheckRsvp(data) {
		log.Print("error")
		return errors.New("Meeting cannot be made"), string(0)
	}

	if err != nil {
		participant := &Participant{
			Name:  participant.Name,
			Email: participant.Email,
			RSVP:  "yes",
		}

		result, _ := collection.InsertOne(ctx, participant)
		log.Print(result, "created")
		return nil, "done"
	}

	resultUpdate, err := collection.UpdateOne(
		ctx,
		bson.M{"email": participant.Email},
		bson.M{
			"$set": bson.M{
				"rsvp": "yes",
			},
		},
	)
	log.Print(resultUpdate, "updated")
	return nil, "success"
}

func SchedueleMeeting(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	response.Header().Set("Access-Control-Request-Method", "POST")

	var meeting Meeting
	_ = json.NewDecoder(request.Body).Decode(&meeting)
	log.Print(meeting.Participants)
	if meeting.Title == "" {
		json.NewEncoder(response).Encode("Please fill all the details")
		return
	}

	for i := 0; i < len(meeting.Participants); i++ {
		err, _ := CreateParticipant(meeting.Participants[i], meeting)
		if err != nil {
			log.Print(err)
			_ = json.NewEncoder(response).Encode("Unable to Scheduele Meeting")
			return
		}
	}

	meet := &Meeting{
		ID:           primitive.NewObjectID(),
		Title:        meeting.Title,
		Participants: meeting.Participants,
		Start:        meeting.Start,
		End:          meeting.End,
		Created:      time.Now(),
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	collection := client.Database("db").Collection("meeting")

	result, _ := collection.InsertOne(ctx, meet)

	var data Meeting

	err := collection.FindOne(ctx, bson.M{"_id": result.InsertedID}).Decode(&data)
	if err != nil {
		log.Print(err)
		json.NewEncoder(response).Encode("No Result")
		return
	}
	json.NewEncoder(response).Encode(data)
}

func CheckRsvp(participant Participant) bool {
	if participant.RSVP == "yes" {
		return false
	}
	return true
}

func GetMeetingUsingId(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	response.Header().Set("Access-Control-Request-Method", "GET")

	var meeting Meeting
	log.Print(meeting.Title)
	log.Print(request.URL.Query().Get("_id"))
	requestid := request.URL.Query().Get("_id")
	collection := client.Database("db").Collection("meeting")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := collection.FindOne(ctx, bson.M{"_id": requestid}).Decode(&meeting)

	if err != nil {
		log.Print(err)
		json.NewEncoder(response).Encode("no result found")
		return
	}

	json.NewEncoder(response).Encode(meeting)
}

func meetingsofparticipants(email string) bool {
	if email == "" {
		return false
	}
	var meets []Meeting
	var meeting []Meeting

	for i := 0; i < len(meeting); i++ {
		for j := 0; j < len(meeting[i].Participants); j++ {
			if meeting[i].Participants[j].Email == email {
				meets = append(meets, meeting[i])
				break
			}
		}
	}
	fmt.Println(meets)
	return true
}

func meetingsofaParticipant(reponse http.ResponseWriter, request *http.Request) {
	email := request.URL.Query().Get("email")
	if meetingsofparticipants(email) {
		json.NewEncoder(reponse).Encode("Meeting Found")
		return
	}
	json.NewEncoder(reponse).Encode("Meeting Not Found")
}

func connectMongo() {
	user := options.Client().ApplyURI("mongodb+srv://admin:admin@cluster0.zh6hh.mongodb.net/db?retryWrites=true&w=majority")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, _ = mongo.Connect(ctx, user)
}

func initiateServer() {
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func main() {
	connectMongo()
	fmt.Println("STARTING THE APPLICATION")

	http.HandleFunc("/meetings", SchedueleMeeting)
	http.HandleFunc("/meetings/{_id}", GetMeetingUsingId)
	http.HandleFunc("/meetings/{email}", meetingsofaParticipant)
	initiateServer()
}
