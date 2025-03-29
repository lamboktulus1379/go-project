package usecase

import (
	"context"
	"encoding/json"

	"my-project/domain/dto"
	"my-project/infrastructure/cache"
	tulushost "my-project/infrastructure/clients/tulustech"
	"my-project/infrastructure/clients/tulustech/models"
	"my-project/infrastructure/logger"
	"my-project/infrastructure/pubsub"
	"my-project/infrastructure/servicebus"
)

type ITestUsecase interface {
	Test(ctx context.Context) dto.TestDto
}

type TestUsecase struct {
	TulusTechHost  tulushost.ITulusHost
	TestPubSub     pubsub.ITestPubSub
	TestServiceBus servicebus.ITestServiceBus
	TestCache      cache.ITestCache
}

type ITulusHost interface {
	GetRandomTyping(ctx context.Context, reqHeader models.ReqHeader) (string, error)
}

func NewTestUsecase(
	tulusTechHost tulushost.ITulusHost,
	testPubSub pubsub.ITestPubSub,
	testServiceBus servicebus.ITestServiceBus,
	testCache cache.ITestCache,
) ITestUsecase {
	return &TestUsecase{
		TulusTechHost:  tulusTechHost,
		TestPubSub:     testPubSub,
		TestServiceBus: testServiceBus,
		TestCache:      testCache,
	}
}

func (testUsecase *TestUsecase) Test(ctx context.Context) dto.TestDto {
	var res dto.TestDto

	res.PubSub = "Not OK"
	res.ServiceBus = "Not OK"

	msg := "Hello"
	byteMsg, err := json.Marshal(msg)
	if err != nil {
		logger.GetLogger().Error("Error while marshalling")
		return res
	}
	publishResponse, err := testUsecase.TestPubSub.Publish(ctx, "topic", byteMsg)
	if err != nil {
		logger.GetLogger().Error("Error while publishing message")
		res.PubSub = err.Error()
		// return res
	}
	logger.GetLogger().WithField("publishResponse", publishResponse).Info("Successfully published")
	res.PubSub = "OK"

	err = testUsecase.TestServiceBus.SendMessage(byteMsg)
	if err != nil {
		logger.GetLogger().Error("Error while publishing message with service bus")
		res.ServiceBus = err.Error()
		// return res
	}
	res.ServiceBus = "OK"

	testUsecase.TestCache.Set(ctx, "test", "test")
	val, err := testUsecase.TestCache.Get(ctx, "test")
	if err != nil {
		logger.GetLogger().Error("Error while getting value from cache")
		res.ServiceBus = "Error while getting value from cache"
		// return res
	}
	res.Cache = val.(string)

	reqHeader := models.ReqHeader{}
	randomTypingRes, err := testUsecase.TulusTechHost.GetRandomTyping(reqHeader)
	if err != nil {
		logger.GetLogger().Error("Error while get random typing")
		res.TulusTech = err.Error()
		// return res
	}
	logger.GetLogger().
		WithField("randomTypingResponse", randomTypingRes).
		Info("Successfully get random typing")
	res.TulusTech = "OK"

	return res
}
