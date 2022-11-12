package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/teris-io/shortid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var collection *mongo.Collection
var usercollection *mongo.Collection
var ctx = context.TODO()
var baseUrl = "https://mysterious-garden-68262.herokuapp.com/"

func init() {
	clientOptons := options.Client().ApplyURI("<mongourl>")
	client, err := mongo.Connect(ctx, clientOptons)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("url_shortner").Collection("urls")
	usercollection = client.Database("url_shortner").Collection("users")
	log.Print("DB connected")
}

type shortenBody struct {
	LongUrl     string `json:"longUrl"`
	UrlCategory string `json:"urlCategory"`
	UserId      string `json:"userId"`
	Api_Key     string `json:"api_key"`
}

type userBody struct {
	UserId string `json:"userId"`
}

type UserDoc struct {
	ID        primitive.ObjectID `bson:"_id"`
	UserId    string             `bson:"userId"`
	Api_Key   string             `bson:"Api_Key"`
	CreatedAt time.Time          `bson:"createdAt"`
	ExpiresAt time.Time          `bson:"expiresAt"`
}

type UrlDoc struct {
	ID          primitive.ObjectID `bson:"_id"`
	UrlCode     string             `bson:"urlCode"`
	LongUrl     string             `bson:"longUrl"`
	ShortUrl    string             `bson:"shortUrl"`
	UrlCategory string             `bson:"urlCategory"`
	CreatedAt   time.Time          `bson:"createdAt"`
	ExpiresAt   time.Time          `bson:"expiresAt"`
	Count       int                `bson:"count"`
}

type customBody struct {
	LongUrl     string `json:"longUrl"`
	CustomCode  string `json:"customCode"`
	UrlCategory string `json:"urlCategory"`
	UserId      string `json:"userId"`
	Api_Key     string `json:"api_key"`
}

func Shorten(c *gin.Context) {
	var body shortenBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, urlErr := url.ParseRequestURI(body.LongUrl)
	if urlErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": urlErr.Error()})
		return
	}

	urlCode, idErr := shortid.Generate()
	if idErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": idErr.Error()})
		return
	}

	var checkApiKey bson.M
	ApiKeyQueryErr := usercollection.FindOne(ctx, bson.D{{"Api_Key", body.Api_Key}}).Decode(&checkApiKey)
	if ApiKeyQueryErr != nil {
		if ApiKeyQueryErr != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": ApiKeyQueryErr.Error()})
			return
		}
	}

	// if len(checkApiKey) > 0 {
	// 	if checkApiKey["userId"] != body.UserId {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("API key not valid for: user_id: %s, api_key: %s, req_user_id: %s", checkApiKey["userId"], checkApiKey["Api_Key"], body.UserId)})
	// 		return
	// 	}
	// }

	if checkApiKey["userId"] == nil || checkApiKey["userId"] != body.UserId {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("API KEY Invalid for : %s", body.UserId)})
		return
	}

	var result bson.M
	queryErr := collection.FindOne(ctx, bson.D{{"urlCode", urlCode}}).Decode(&result)

	if queryErr != nil {
		if queryErr != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": queryErr.Error()})
			return
		}
	}

	if len(result) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Code in use: %s", urlCode)})
		return
	}

	var date = time.Now()
	var expires = date.AddDate(0, 0, 5)
	var newUrl = baseUrl + urlCode
	var docId = primitive.NewObjectID()
	var count = 0

	newDoc := &UrlDoc{
		ID:          docId,
		UrlCode:     urlCode,
		LongUrl:     body.LongUrl,
		UrlCategory: body.UrlCategory,
		ShortUrl:    newUrl,
		Count:       count,
		CreatedAt:   time.Now(),
		ExpiresAt:   expires,
	}

	_, err := collection.InsertOne(ctx, newDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"newUrl":      newUrl,
		"urlCategory": body.UrlCategory,
		"expires":     expires.Format("2006-01-02 15:04:05"),
		"db_id":       docId,
		"count":       count,
		"userid":      checkApiKey["userId"],
	})
}

func Redirect(c *gin.Context) {
	code := c.Param("code")
	var result bson.M
	queryErr := collection.FindOneAndUpdate(ctx, bson.D{{"urlCode", code}}, bson.D{{
		Key:   "$inc",
		Value: bson.D{{"count", 1}},
	}}).Decode(&result)

	if queryErr != nil {
		if queryErr == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No URL with code: %s", code)})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": queryErr.Error()})
			return
		}
	}
	log.Print(result["longUrl"])
	var longUrl = fmt.Sprint(result["longUrl"])
	c.Redirect(http.StatusPermanentRedirect, longUrl)
}

func Custom(c *gin.Context) {
	var body customBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, urlErr := url.ParseRequestURI(body.LongUrl)
	if urlErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}
	var length = len(body.CustomCode)
	if length < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Custom code should be more than 3 characters"})
		return
	}

	var checkApiKey bson.M
	ApiKeyQueryErr := usercollection.FindOne(ctx, bson.D{{"Api_Key", body.Api_Key}}).Decode(&checkApiKey)
	if ApiKeyQueryErr != nil {
		if ApiKeyQueryErr != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": ApiKeyQueryErr.Error()})
			return
		}
	}

	if checkApiKey["userId"] == nil || checkApiKey["userId"] != body.UserId {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("API KEY Invalid for : %s", body.UserId)})
		return
	}

	var result bson.M
	queryErr := collection.FindOne(ctx, bson.D{{"urlCode", body.CustomCode}}).Decode(&result)

	if queryErr != nil {
		if queryErr != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": queryErr.Error()})
			return
		}
	}

	if len(result) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Code in use: %s", body.CustomCode)})
		return
	}

	var date = time.Now()
	var expires = date.AddDate(0, 0, 5)
	var newUrl = baseUrl + body.CustomCode
	var docId = primitive.NewObjectID()
	var count = 0

	newDoc := &UrlDoc{
		ID:          docId,
		UrlCode:     body.CustomCode,
		LongUrl:     body.LongUrl,
		ShortUrl:    newUrl,
		UrlCategory: body.UrlCategory,
		Count:       count,
		CreatedAt:   time.Now(),
		ExpiresAt:   expires,
	}

	_, err := collection.InsertOne(ctx, newDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"newUrl":  newUrl,
		"expires": expires.Format("2006-01-02 15:04:05"),
		"db_id":   docId,
		"count":   count,
		"userid":  checkApiKey["userId"],
	})
}

func GenerateUser(c *gin.Context) {
	userId, idErr := shortid.Generate()
	if idErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": idErr.Error()})
		return
	}

	api_key, idErr := shortid.Generate()
	if idErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": idErr.Error()})
		return
	}

	var result bson.M
	queryErr := usercollection.FindOne(ctx, bson.D{{"userId", userId}}).Decode(&result)

	if queryErr != nil {
		if queryErr != mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": queryErr.Error()})
			return
		}
	}

	if len(result) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("user id in use: %s", userId)})
		return
	}

	var docId = primitive.NewObjectID()

	newDoc := &UserDoc{
		ID:        docId,
		UserId:    userId,
		Api_Key:   api_key,
		CreatedAt: time.Now(),
	}

	_, err := usercollection.InsertOne(ctx, newDoc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"User_id": userId,
		"api_key": api_key,
		"messgae": "use following api key for auth",
	})

}

func GetUserAPIKey(c *gin.Context) {
	var body userBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var findUserAPIKey bson.M
	ApiKeyQueryErr := usercollection.FindOne(ctx, bson.D{{"userId", body.UserId}}).Decode(&findUserAPIKey)
	if ApiKeyQueryErr != nil {
		if ApiKeyQueryErr == mongo.ErrNoDocuments {
			c.JSON(http.StatusInternalServerError, gin.H{"error": ApiKeyQueryErr.Error()})
			return
		}
	}

	if len(findUserAPIKey) > 0 {
		if findUserAPIKey["Api_Key"] != nil {
			c.JSON(http.StatusOK, gin.H{
				"API_KEY": findUserAPIKey["Api_Key"],
			})
			return
		}
	}

}

func GetStats(c *gin.Context) {
	code := c.Param("code")
	var result bson.M
	queryErr := collection.FindOne(ctx, bson.D{{"urlCode", code}}).Decode(&result)

	if queryErr != nil {
		if queryErr == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("No URL with code: %s", code)})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": queryErr.Error()})
			return
		}
	}
	log.Print(result["longUrl"])
	var longUrl = fmt.Sprint(result["longUrl"])
	var count = fmt.Sprint(result["count"])
	var category = fmt.Sprint(result["urlCategory"])

	c.JSON(http.StatusOK, gin.H{
		"Long Url": longUrl,
		"Clicks":   count,
		"category": category,
	})
}
