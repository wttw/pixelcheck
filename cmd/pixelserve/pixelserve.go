package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	flag "github.com/spf13/pflag"
	"github.com/wttw/pixelcheck"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

const appName = "pixelserve"

func main() {
	var configFile string
	helpRequest := false
	flag.StringVar(&configFile, "config", "pixelcheck.json", "Alternate configuration file")
	flag.BoolVarP(&helpRequest, "help", "h", false, "Display brief help")

	flag.Parse()
	if helpRequest {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", appName)
		flag.PrintDefaults()
		os.Exit(0)
	}
	// Get our configuration
	c := pixelcheck.New(configFile)

	// A database would be nice
	db, err := pgxpool.Connect(context.Background(), c.DBConn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}

	// Create the tables we need, if needed

	// language=SQL
	_, err = db.Exec(context.Background(),
		`create table if not exists image (id integer primary key generated always as identity,
 				path text unique,
 				url text,
 				file text not null,
 				created timestamptz not null default current_timestamp)`,
	)
	if err != nil {
		log.Fatal(err)
	}

	// language=SQL
	_, err = db.Exec(context.Background(),
		`create table if not exists mail (id integer primary key generated always as identity,
				recipient text not null,
				sender text not null,
				body text not null,
				template text not null,
				image integer not null references image,
				note text not null,
				sent timestamptz not null default current_timestamp)`,
	)
	if err != nil {
		log.Fatal(err)
	}
	// language=SQL
	_, err = db.Exec(context.Background(),
		`create table if not exists load(id integer primary key generated always as identity,
             image integer not null references image,
             peer inet not null,
             port integer not null,
             proto text not null,
             headers jsonb not null,
             uri text not null,
             loaded timestamptz not null default current_timestamp)`,
	)
	if err != nil {
		log.Fatal(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var imageID int
		var imageFile string
		// language=SQL
		err := db.QueryRow(r.Context(), `select id, file from image where path=$1`, r.URL.Path).Scan(&imageID, &imageFile)
		if err != nil {
			if err == pgx.ErrNoRows {
				log.Printf("Received request for non-existent image %s", r.URL.Path)
			} else {
				log.Printf("Error retrieving %s from DB: %s", r.URL.Path, err)
			}
			http.Error(w, "no such file", http.StatusNotFound)
			return
		}

		var loadID int
		addr, port, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Bad remoteaddr: %s", r.RemoteAddr)
		}

		// language=SQL
		err = db.QueryRow(r.Context(), `insert into load (image, peer, port, proto, headers, uri) VALUES ($1, $2, $3, $4, $5, $6) returning id`, imageID, addr, port, r.Proto, r.Header, r.URL.String()).Scan(&loadID) //nolint:lll
		if err != nil {
			log.Printf("Failed to record load: %s", err)
		}
		log.Printf("Returning %s %d for request %d %s", imageFile, imageID, loadID, r.URL)
		http.ServeFile(w, r, filepath.Join(c.ImageDir, imageFile))
	})
	log.Printf("Listening on %s", c.Listen)
	if c.Cert != "" {
		err = http.ListenAndServeTLS(c.Listen, c.Cert, c.Key, handler)
	} else {
		err = http.ListenAndServe(c.Listen, handler)
	}
	if err != nil {
		log.Fatalf("Server exiting: %s", err)
	}
}
