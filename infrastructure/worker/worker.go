package worker

import (
	"database/sql"
	"fmt"
	"log"
	"my-project/infrastructure/logger"
	"sync"
	"time"

	"my-project/domain/model"
)

func PooledWorkError(allData []model.Project, db *sql.DB) {
	start := time.Now()
	var wg sync.WaitGroup
	workerPoolSize := 100

	dataCh := make(chan model.Project, workerPoolSize)
	errors := make(chan error, 100)

	for i := 0; i < workerPoolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for data := range dataCh {
				process(data, db, errors)
			}
		}()
	}

	for i := range allData {
		dataCh <- allData[i]
	}

	close(dataCh)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case err := <-errors:
				fmt.Println("finished with error:", err.Error())
			case <-time.After(time.Second * 1):
				fmt.Println("Timeout: errors finished")
				return
			}
		}
	}()

	defer close(errors)
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("Took ===============> %s\n", elapsed)
}

func process(data model.Project, db *sql.DB, errors chan<- error) {
	fmt.Printf("Start processing %s\n", data.Name)
	time.Sleep(100 * time.Millisecond)

	if data.Name == "" {
		errors <- fmt.Errorf("error on job %v", data.Name)
	} else {
		result, err := db.Exec("INSERT INTO project (name, description) VALUES($1, $2)", data.Name, data.Description)
		if err != nil {
			log.Printf("An error occurred %v", err)
		}
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				logger.GetLogger().Error(err)
			}
		}(db)
		id, err := result.LastInsertId()
		if err != nil {
			log.Printf("Fail to get last inserted id")
		}

		fmt.Printf("Finish processing %s With %d\n", data.Name, id)
	}
}
