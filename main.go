package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/uploader"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/rs/cors"
)

type Media struct {
	Id       string `db:"media_id"`
	PostId   string `db:"post_id"`
	ImageUrl string `db:"image_url"`
}

var dbClient *sqlx.DB

func uploadImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postId := vars["id"]

	log.Println("Started upload for post-id ", postId)

	exists := checkIfMediaExists(postId)
	if exists {
		writeResponse(w, http.StatusBadRequest, "media-exists")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	f, err := os.OpenFile(handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, _ = io.Copy(f, file)

	cld, _ := cloudinary.NewFromParams("dcnha5nlt", "936397336393941", "S2EQnLnlWAZh1eohI_zoOe0JlI0")
	resp, err := cld.Upload.Upload(context.Background(), handler.Filename, uploader.UploadParams{})

	if err != nil {
		log.Println("Error while uploading image to cloudinary for post-id ", postId, err.Error())
		writeResponse(w, http.StatusInternalServerError, "image-cloudinary-upload-error")
		return
	}

	imageUrl := resp.SecureURL
	log.Println("Image successfully uploaded to cloudinary for post-id ", postId)
	log.Println("Image URL is ", imageUrl)
	os.Remove(handler.Filename)

	media := saveMedia(Media{PostId: postId, ImageUrl: imageUrl})

	if media == nil {
		writeResponse(w, http.StatusInternalServerError, "image-db-save-error")
	} else {
		writeResponse(w, http.StatusOK, media)
	}
}

func checkIfMediaExists(postId string) bool {
	query := "SELECT media_id, post_id, image_url FROM media WHERE post_id = ?"
	var media Media
	err := dbClient.Get(&media, query, postId)

	if err == nil {
		log.Println("Media for post-id ", postId, " already exists.")
		return true
	}

	return false
}

func getMediaByImageId(w http.ResponseWriter, r *http.Request) {
	query := "SELECT media_id, post_id, image_url FROM media WHERE media_id = ?"
	var media Media
	vars := mux.Vars(r)
	postId := vars["id"]
	err := dbClient.Get(&media, query, postId)

	if err != nil {
		log.Println("Media for post-id ", postId, " already exists.")
		writeResponse(w, http.StatusNotFound, "media-not-found")
	} else {
		writeResponse(w, http.StatusOK, media)
	}
}

func saveMedia(m Media) *Media {
	query := "INSERT INTO media (post_id, image_url) VALUES (?, ?)"
	result, err := dbClient.Exec(query, m.PostId, m.ImageUrl)

	if err != nil {
		log.Println("Error while inserting media into db for post-id ", m.PostId, err.Error())
		return nil
	}
	id, err := result.LastInsertId()

	if err != nil {
		log.Println("Error while getting last inserted media-id from db for post-id ", m.PostId, err.Error())
		return nil
	}

	m.Id = strconv.FormatInt(id, 10)
	return &m
}

func writeResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		panic(err)
	}
}

func createDbConnection() *sqlx.DB {
	dbAddr := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s",
		"root", "root", "media-service-mysql", "3306", "NistagramMediaService",
	)

	client, err := sqlx.Open("mysql", dbAddr)
	if err != nil {
		panic(err)
	}

	client.SetConnMaxLifetime(time.Minute * 3)
	client.SetMaxOpenConns(10)
	client.SetMaxIdleConns(10)

	return client
}

func main() {
	dbClient = createDbConnection()
	router := mux.NewRouter()

	router.HandleFunc("/api/media/{id}", uploadImage).Methods(http.MethodPost)
	router.HandleFunc("/api/media/{id}", getMediaByImageId).Methods((http.MethodGet))
	router.HandleFunc("/api/media", func(w http.ResponseWriter, r *http.Request) {
		writeResponse(w, http.StatusOK, "Hello world")
	}).Methods(http.MethodGet)

	// headersOk := handlers.AllowedHeaders([]string{"X-Requested-With"})
	// originsOk := handlers.AllowedOrigins([]string{os.Getenv("ORIGIN_ALLOWED")})
	// methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	log.Printf("Started nistagram-media-service on port %s", "8000")
	// log.Fatal(http.ListenAndServe("0.0.0.0:8000", handlers.CORS(originsOk, headersOk, methodsOk)(router)))

	handler := cors.Default().Handler(router)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200"},
		AllowCredentials: true,
	})

	// Insert the middleware
	handler = c.Handler(handler)
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", handler))

}
