package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "os"
	_ "strconv"

	"github.com/Tnze/go-mc/save"
	_ "github.com/Tnze/go-mc/save/region"
	"github.com/gorilla/mux"
)

func apiAddChunkHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	dname := params["dim"]
	sname := params["server"]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errmsg := fmt.Sprintf("Error reading request: %s", err)
		w.Write([]byte(errmsg))
		log.Print(errmsg)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var col save.Column
	err = col.Load(body)
	if err != nil {
		errmsg := fmt.Sprintf("Error parsing chunk data: %s", err)
		w.Write([]byte(errmsg))
		log.Print(errmsg)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tag, err := dbpool.Exec(context.Background(), `
		insert into chunks (x, z, data, dim)
		values ($1, $2, $3
			(select dimensions.id 
			 from dimensions 
			 join servers on servers.id = dimensions.server 
			 where servers.name = $4 and dimensions.name = $5))`,
		col.Level.PosX, col.Level.PosZ, body, sname, dname)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
		return
	}
	log.Print("Submitted chunk ", col.Level.PosX, col.Level.PosZ, " server ", sname, " dimension ", dname)
	if tag.RowsAffected() != 1 {
		log.Print("Rows affected ", tag.RowsAffected())
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Chunk %d:%d of %s:%s submitted. Thank you for your contribution!\n", col.Level.PosX, col.Level.PosZ, sname, dname)))
	return
}

func apiAddRegionHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	return
	// params := mux.Vars(r)
	// dids := params["did"]
	// did, err := strconv.Atoi(dids)
	// if err != nil {
	// 	errmsg := fmt.Sprintf("Bad dim id: %s", err)
	// 	w.Write([]byte(errmsg))
	// 	log.Print(errmsg)
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	// body, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	errmsg := fmt.Sprintf("Error reading request: %s", err)
	// 	w.Write([]byte(errmsg))
	// 	log.Print(errmsg)
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	// f, err := os.CreateTemp("", "upload")
	// if err != nil {
	// 	errmsg := fmt.Sprintf("Error creating region file: %s", err)
	// 	w.Write([]byte(errmsg))
	// 	log.Print(errmsg)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// defer os.Remove(f.Name())
	// if n, err := f.Write(body); err != nil || n != len(body) {
	// 	errmsg := fmt.Sprintf("Error writing region file: %s", err)
	// 	w.Write([]byte(errmsg))
	// 	log.Print(errmsg)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// if err := f.Close(); err != nil {
	// 	errmsg := fmt.Sprintf("Error closing region file: %s", err)
	// 	w.Write([]byte(errmsg))
	// 	log.Print(errmsg)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// region, err := region.Open(f.Name())
	// if err != nil {
	// 	errmsg := fmt.Sprintf("Error opening region file: %s", err)
	// 	w.Write([]byte(errmsg))
	// 	log.Print(errmsg)
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }
	// for x := 0; x < 32; x++ {
	// 	for z := 0; z < 32; z++ {
	// 		if !region.ExistSector(x, z) {
	// 			continue
	// 		}
	// 		data, err := region.ReadSector(x, z)
	// 		if err != nil {
	// 			log.Printf("Read sector (%d.%d) error: %v", x, z, err)
	// 		}
	// 		var col save.Column
	// 		col.Load(data)
	// 		tag, err := dbpool.Exec(context.Background(), `insert into chunks (dim, x, z, data) values ($1, $2, $3, $4)`, did, col.Level.PosX, col.Level.PosZ, data)
	// 		if err != nil {
	// 			w.WriteHeader(http.StatusInternalServerError)
	// 			log.Print(err.Error())
	// 			return
	// 		}
	// 		// log.Print("Submitted chunk ", col.Level.PosX, col.Level.PosZ)
	// 		if tag.RowsAffected() != 1 {
	// 			log.Print("Rows affected ", tag.RowsAffected())
	// 		}
	// 	}
	// }
	// region.Close()
	// w.WriteHeader(http.StatusOK)
	// w.Write([]byte(fmt.Sprintf("Region submitted. Thank you for your contribution!\n")))
	// return
}
