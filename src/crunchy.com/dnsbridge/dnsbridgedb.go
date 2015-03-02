/*
 Copyright 2015 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package dnsbridge

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
)

type BridgeEntry struct {
	ID         string
	Name       string
	IPAddress  string
	CreateDate string
}

var pguser = "bridge"
var pgpass = "bridge"
var pghost = "127.0.0.1"
var pgport = "5432"
var db = "bridge"

//var dbConn *sql.DB

func Get(id string) (BridgeEntry, error) {
	bridge := BridgeEntry{}
	dbConn, err1 := sql.Open("postgres", "sslmode=disable user="+pguser+" password="+pgpass+" host="+pghost+" port="+pgport+" dbname="+db)
	defer dbConn.Close()
	if err1 != nil {
		return bridge, err1
	}

	fmt.Printf("Get called with id=%s\n", id)
	queryStr := fmt.Sprintf("select id, name, ipaddress, to_char(createdt, 'MM-DD-YYYY HH24:MI:SS') from bridge where id='%s'", id)
	fmt.Printf("Get query %s\n", queryStr)

	rows, err := dbConn.Query("select id, name, ipaddress, to_char(createdt, 'MM-DD-YYYY HH24:MI:SS') from bridge where id=$1", &id)
	if err != nil {
		return bridge, err
	}
	bridge2 := BridgeEntry{"", "", "", ""}
	for rows.Next() {
		if err := rows.Scan(&bridge2.ID, &bridge2.Name, &bridge2.IPAddress, &bridge2.CreateDate); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("Name is %s\n", bridge2.Name)
	}

	if err := rows.Err(); err != nil {
		fmt.Println(err.Error())
	}
	return bridge2, nil

	/*
		rows, err := dbConn.Query(queryStr)
		if err != nil {
			return bridge, err
		}

		for rows.Next() {
			bridge2 := BridgeEntry{}
			if  err = rows.Scan(&bridge2.ID, &bridge2.Name, &bridge2.IPAddress, &bridge2.CreateDate); err != nil {
				return bridge2, err
			}
			return bridge2, nil
		}
	*/

	//if you get here it means no rows were found which can happen
	//return bridge, nil
}

func Insert(b BridgeEntry) error {
	dbConn, err := sql.Open("postgres", "sslmode=disable user="+pguser+" host="+pghost+" port="+pgport+" dbname="+db)
	if err != nil {
		fmt.Printf(err.Error())
		return err
	}
	defer dbConn.Close()
	fmt.Println("Insert called")

	queryStr := fmt.Sprintf("insert into bridge ( id, name, ipaddress, createdt) values ( '%s', '%s', '%s', now()) ", b.ID, b.Name, b.IPAddress)

	fmt.Println(queryStr)
	rc := dbConn.QueryRow(queryStr)
	switch {
	case rc != nil:
		return errors.New("could not insert into bridge table...error")
	}

	return nil
}

func Delete(id string) error {
	dbConn, err := sql.Open("postgres", "sslmode=disable user="+pguser+" host="+pghost+" port="+pgport+" dbname="+db)
	if err != nil {
		return err
	}
	defer dbConn.Close()

	queryStr := fmt.Sprintf("delete from bridge where id = '%s'", id)

	fmt.Println(queryStr)
	rc := dbConn.QueryRow(queryStr)
	switch {
	case rc != nil:
		return errors.New("could not delete bridge entry...error")
	}
	return nil
}
