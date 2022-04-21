/*
	WebChunk, web server for block game maps
	Copyright (C) 2022 Maxim Zhuchkov

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published
	by the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.

	Contact me via mail: q3.max.2011@yandex.ru or Discord: MaX#6717
*/

package main

import (
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
)

var (
	dimNameRegexp  = regexp.MustCompile(`[\-a-zA-Z0-9.]+`)
	dimAliasRegexp = dimNameRegexp
)

func dimensionHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	wname := params["world"]
	dname := params["dim"]
	world, s, err := getWorldStorage(wname)
	if err != nil {
		plainmsg(w, r, plainmsgColorRed, "Error getting storage interface by world name: "+err.Error())
		return
	}
	if s == nil || world == nil {
		plainmsg(w, r, plainmsgColorRed, "World not found")
		return
	}
	dim, err := s.GetDimension(wname, dname)
	if err != nil {
		plainmsg(w, r, plainmsgColorRed, "Error getting dimension from storage: "+err.Error())
		return
	}
	if dim == nil {
		plainmsg(w, r, plainmsgColorRed, "Dimension not found")
		return
	}
	basicLayoutLookupRespond("dim", w, r, map[string]interface{}{"Dim": dim, "World": world})
}

func apiAddDimension(w http.ResponseWriter, r *http.Request) (int, string) {
	if r.ParseMultipartForm(0) != nil {
		return 400, "Unable to parse form parameters"
	}
	name := r.FormValue("name")
	if !dimNameRegexp.Match([]byte(name)) {
		return 400, "Invalid dimension name"
	}
	alias := r.FormValue("alias")
	if !dimAliasRegexp.Match([]byte(alias)) {
		return 400, "Invalid dimension alias"
	}
	wname := r.FormValue("world")
	if !worldNameRegexp.Match([]byte(wname)) {
		return 400, "Invalid world name"
	}
	_, s, err := getWorldStorage(wname)
	if err != nil {
		return 500, "Error getting world storage: " + err.Error()
	}
	if s == nil {
		return 404, "World does not exist"
	}
	dim, err := s.AddDimension(wname, name, alias)
	if err != nil {
		return 500, "Failed to add dimension: " + err.Error()
	}
	setContentTypeJson(w)
	return marshalOrFail(200, dim)
}

func apiListDimensions(w http.ResponseWriter, r *http.Request) (int, string) {
	if r.ParseForm() != nil {
		return 400, "Unable to parse form parameters"
	}
	dims, err := listDimensions(r.Form.Get("world"))
	if err != nil {
		return 500, "Failed to list dimensions: " + err.Error()
	}
	setContentTypeJson(w)
	return marshalOrFail(200, dims)
}
