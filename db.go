package main

import (
	"context"
	"math/rand"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	_ "github.com/joho/godotenv/autoload"
)

var (
	database = db.Database("go")
	opts     = options.Update().SetUpsert(true)
)

func DBinit() *mongo.Client {
	db, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("MONGO_DB_URI")))
	if err != nil {
		panic(err)
	}
	return db
}

func genRandVoteId() string {
	var id string
	for i := 0; i < 10; i++ {
		id += string(rune(65 + rand.Intn(25))) // A-Z
	}
	return id
}

func genRandTeamId() string {
	var id string
	for i := 0; i < 10; i++ {
		id += string(rune(65 + rand.Intn(25))) // A-Z
	}
	return id
}

var db = DBinit()

type Team struct {
	ID      string `json:"id" bson:"_id"`
	Name    string `json:"name" bson:"name"`
	Project string `json:"project" bson:"project"`
}

type Vote struct {
	ID       string  `json:"id" bson:"_id"`
	TeamName string  `json:"team_name" bson:"team_name"`
	IsJudge  bool    `json:"is_judge" bson:"is_judge"`
	Rating   float64 `json:"rating" bson:"rating"`
	IpAddr   string  `json:"ip_addr" bson:"ip_addr"`
	Feedback string  `json:"feedback" bson:"feedback"`
	Sender   string  `json:"sender" bson:"sender"`
}

func GetTeam(id string) (Team, error) {
	var team Team
	err := database.Collection("teams").FindOne(context.TODO(), bson.M{"_id": id}).Decode(&team)
	return team, err
}

func GetTeamByName(name string) (Team, error) {
	var team Team
	err := database.Collection("teams").FindOne(context.TODO(), bson.M{"name": name}).Decode(&team)
	return team, err
}

func GetTeams() ([]Team, error) {
	var teams []Team
	cursor, err := database.Collection("teams").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

func GetVotes() ([]Vote, error) {
	var votes []Vote
	cursor, err := database.Collection("votes").Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &votes); err != nil {
		return nil, err
	}
	return votes, nil
}

func (team Team) InsertTeam() error {
	team.ID = genRandTeamId()
	_, err := database.Collection("teams").InsertOne(context.TODO(), team)
	return err
}

func (vote Vote) InsertVote() error {
	vote.ID = genRandVoteId()
	_, err := database.Collection("votes").InsertOne(context.TODO(), vote)
	return err
}

func HasIpVoted(ip string, teamName string) bool {
	var vote Vote
	err := database.Collection("votes").FindOne(context.TODO(), bson.M{"ip_addr": ip, "team_name": teamName}).Decode(&vote)
	return err == nil
}

func (team Team) Delete() error {
	_, err := database.Collection("teams").DeleteOne(context.TODO(), bson.M{"_id": team.ID})
	return err
}

func DeleteTeam(name string) error {
	_, err := database.Collection("teams").DeleteOne(context.TODO(), bson.M{"name": name})
	return err
}

func (team Team) Update() error {
	_, err := database.Collection("teams").UpdateOne(context.TODO(), bson.M{"_id": team.ID}, bson.M{"$set": team}, opts)
	return err
}

func calcAvgRating(teamName string) (float64, error) {
	var votes []Vote
	cursor, err := database.Collection("votes").Find(context.TODO(), bson.M{"team_name": teamName})
	if err != nil {
		return 0, err
	}
	if err = cursor.All(context.TODO(), &votes); err != nil {
		return 0, err
	}

	var total float64
	for _, vote := range votes {
		total += vote.Rating
	}
	return total / float64(len(votes)), nil
}

func (team Team) GetAvgRating() (float64, error) {
	return calcAvgRating(team.Name)
}

func (team Team) GetVotes() ([]Vote, error) {
	var votes []Vote
	cursor, err := database.Collection("votes").Find(context.TODO(), bson.M{"team_name": team.Name})
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &votes); err != nil {
		return nil, err
	}
	return votes, nil
}
