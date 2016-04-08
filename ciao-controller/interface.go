/*
// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
*/

package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func tenantDebug(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "GET" {
		t, err := template.New("t").ParseFiles("tenantDebug.gtpl")
		if err != nil {
			panic(err)
		}
		data := *identityURL

		err = t.ExecuteTemplate(w, "tenantDebug.gtpl", data)
		if err != nil {
			panic(err)
		}
	}
}

func login(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "GET" {
		t, err := template.New("t").ParseFiles("login.gtpl")
		if err != nil {
			panic(err)
		}
		data := *identityURL

		err = t.ExecuteTemplate(w, "login.gtpl", data)
		if err != nil {
			panic(err)
		}
	}
}

func framestats(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "GET" {
		t, err := template.New("t").ParseFiles("framestats.gtpl")
		if err != nil {
			panic(err)
		}
		err = t.ExecuteTemplate(w, "framestats.gtpl", nil)
		if err != nil {
			panic(err)
		}
	}
}

func stats(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "GET" {
		t, err := template.New("t").ParseFiles("stats.gtpl")
		if err != nil {
			panic(err)
		}
		err = t.ExecuteTemplate(w, "stats.gtpl", nil)
		if err != nil {
			panic(err)
		}
	} else if r.Method == "POST" {
		action := r.PostFormValue("admin_action")
		switch action {
		case "deleteAll", "cleanAll":
			instances, err := context.ds.GetAllInstances()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for index := range instances {
				var err error
				id := instances[index].Id
				if action == "deleteAll" {
					if instances[index].State != "pending" {
						err = context.deleteInstance(id)
					}
				} else if action == "cleanAll" {
					if instances[index].State == "pending" {
						err = context.ds.DeleteInstance(id)
					}
				}
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

		case "clearEventLog":
			context.ds.ClearLog()
		case "evacuate":
			val := r.PostFormValue("node_ids")
			nodeIds := strings.Fields(val)
			for index := range nodeIds {
				context.evacuateNode(nodeIds[index])
			}
		case "delete", "clean", "restart", "stop":
			val := r.PostFormValue("instances_ids")
			instanceIDs := strings.Fields(val)
			for index := range instanceIDs {
				var err error
				instanceID := instanceIDs[index]
				if action == "delete" {
					err = context.deleteInstance(instanceID)
				} else if action == "clean" {
					err = context.ds.DeleteInstance(instanceID)
				} else if action == "stop" {
					err = context.stopInstance(instanceID)
				} else if action == "restart" {
					err = context.restartInstance(instanceID)
				}
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		http.Redirect(w, r, "/stats", http.StatusFound)
	}
}

func getBatchFrameSummaryStats(w http.ResponseWriter, r *http.Request, context *controller) {
	stats, err := context.ds.GetBatchFrameSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	b, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func getBatchFrameStats(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "POST" {
		label := r.PostFormValue("batch_id")

		stats, err := context.ds.GetFrameStatistics(label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		b, err := json.Marshal(stats)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	} else if r.Method == "GET" {
	}
}

func getFrameStats(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "POST" {
		label := r.PostFormValue("batch_id")

		stats, err := context.ds.GetBatchFrameStatistics(label)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		b, err := json.Marshal(stats)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	} else if r.Method == "GET" {
	}
}

func getEventLog(w http.ResponseWriter, r *http.Request, context *controller) {
	logEntries, err := context.ds.GetEventLog()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(logEntries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func getNodeSummary(w http.ResponseWriter, r *http.Request, context *controller) {
	summary, err := context.ds.GetNodeSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(summary)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func getNodeStats(w http.ResponseWriter, r *http.Request, context *controller) {
	end := time.Now().UTC()
	start := end.Add(-20 * time.Minute)

	statsRows, err := context.ds.GetNodeStats(start, end)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(statsRows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	// hack - clean up the stats so it doesn't get too big
	_ = context.ds.ClearNodeStats(start)
}

func getInstances(w http.ResponseWriter, r *http.Request, context *controller) {
	instances, err := context.ds.GetAllInstances()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	b, err := json.Marshal(instances)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func getWorkloads(w http.ResponseWriter, r *http.Request, context *controller) {
	workloads, err := context.ds.GetWorkloads()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	b, err := json.Marshal(workloads)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func getCNCI(w http.ResponseWriter, r *http.Request, context *controller) {
	cncis, err := context.ds.GetTenantCNCISummary("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(cncis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func getNodes(w http.ResponseWriter, r *http.Request, context *controller) {
	nodes, err := context.ds.GetNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, err := json.Marshal(nodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func workload(w http.ResponseWriter, r *http.Request, context *controller) {
	if r.Method == "POST" {
		token := r.PostFormValue("token")

		trace, err := strconv.ParseBool(r.PostFormValue("trace"))
		if err != nil {
			http.Error(w, "Bogus Trace value", http.StatusInternalServerError)
			return
		}

		label := r.PostFormValue("label")
		r.ParseForm()
		var tenantID string
		var workloadID string
		var numInstances int

		if len(r.Form["workload_id"]) > 0 {
			workloadID = r.Form["workload_id"][0]
		} else {
			http.Error(w, "Missing workload", http.StatusInternalServerError)
			return
		}

		if len(r.Form["tenant_id"]) > 0 {
			tenantID = r.Form["tenant_id"][0]
		} else {
			http.Error(w, "Missing tenant ID", http.StatusInternalServerError)
			return
		}

		numInstances, _ = strconv.Atoi(r.Form["num_instances"][0])

		ok := context.id.validate(token, tenantID, "user")
		if ok {
			_, err := context.startWorkload(workloadID, tenantID, numInstances, trace, label)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "Token Validation Failed", http.StatusInternalServerError)
			return
		}
	}
}

func createDebugInterface(context *controller) {
	http.HandleFunc("/workload", func(w http.ResponseWriter, r *http.Request) {
		workload(w, r, context)
	})
	http.HandleFunc("/tenantDebug", func(w http.ResponseWriter, r *http.Request) {
		tenantDebug(w, r, context)
	})
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats(w, r, context)
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		login(w, r, context)
	})
	http.HandleFunc("/framestats", func(w http.ResponseWriter, r *http.Request) {
		framestats(w, r, context)
	})
	http.HandleFunc("/getNodeStats", func(w http.ResponseWriter, r *http.Request) {
		getNodeStats(w, r, context)
	})
	http.HandleFunc("/getInstances", func(w http.ResponseWriter, r *http.Request) {
		getInstances(w, r, context)
	})
	http.HandleFunc("/getEventLog", func(w http.ResponseWriter, r *http.Request) {
		getEventLog(w, r, context)
	})
	http.HandleFunc("/getNodeSummary", func(w http.ResponseWriter, r *http.Request) {
		getNodeSummary(w, r, context)
	})
	http.HandleFunc("/getWorkloads", func(w http.ResponseWriter, r *http.Request) {
		getWorkloads(w, r, context)
	})
	http.HandleFunc("/getCNCI", func(w http.ResponseWriter, r *http.Request) {
		getCNCI(w, r, context)
	})
	http.HandleFunc("/getFrameStats", func(w http.ResponseWriter, r *http.Request) {
		getFrameStats(w, r, context)
	})
	http.HandleFunc("/getBatchFrameSummaryStats", func(w http.ResponseWriter, r *http.Request) {
		getBatchFrameSummaryStats(w, r, context)
	})
	http.HandleFunc("/getBatchFrameStats", func(w http.ResponseWriter, r *http.Request) {
		getBatchFrameStats(w, r, context)
	})
	http.HandleFunc("/getNodes", func(w http.ResponseWriter, r *http.Request) {
		getNodes(w, r, context)
	})
	httpPort := ":" + strconv.Itoa(*port)
	log.Fatal(http.ListenAndServeTLS(httpPort, *httpsCAcert, *httpsKey, nil))
}
