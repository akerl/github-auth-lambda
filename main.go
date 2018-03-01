package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("Hello world")

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("%+v\n", request),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
