package persistence

import (
	"context"
	"database/sql"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"my-project/domain/model"
	"my-project/infrastructure/logger"
	"my-project/infrastructure/worker"
)

type ITestRepository interface {
	Test(ctx context.Context) ([]model.Project, error)
}

type TestRepository struct {
	mongoDb    *mongo.Client
	PostgresDB *sql.DB
}

func (t *TestRepository) Test(ctx context.Context) ([]model.Project, error) {
	myProjects := []model.Project{
		{
			Name:        "Project 1",
			Description: "Description 1",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	worker.PooledWorkError(myProjects, t.PostgresDB)
	collection := t.mongoDb.Database("my_project").Collection("projects")
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while fetching data")
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error while closing cursor")
		}
	}(cursor, ctx)

	var projects []model.Project
	for cursor.Next(ctx) {
		var project model.Project
		err := cursor.Decode(&project)
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error while decoding")
		}
		projects = append(projects, project)
	}
	return projects, nil
}

func NewTestRepository(db *mongo.Client, postgresDB *sql.DB) ITestRepository {
	return &TestRepository{mongoDb: db, PostgresDB: postgresDB}
}
