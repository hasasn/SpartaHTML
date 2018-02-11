package main

import (
	"context"
	"fmt"
	"net/http"

	sparta "github.com/mweagle/Sparta"
	spartaCF "github.com/mweagle/Sparta/aws/cloudformation"
	spartaAPIG "github.com/mweagle/Sparta/aws/events"
	gocf "github.com/mweagle/go-cloudformation"
	"github.com/sirupsen/logrus"
)

type lambhackResponse struct {
	Message string
	Request spartaAPIG.APIGatewayRequest
}

////////////////////////////////////////////////////////////////////////////////
// lambhack event handler
func lambhack(ctx context.Context,
	gatewayEvent spartaAPIG.APIGatewayRequest) (lambhackResponse, error) {

	logger, loggerOk := ctx.Value(sparta.ContextKeyLogger).(*logrus.Logger)
	if loggerOk {
		logger.Info("Lambhack structured log message")
	}

	command := gatewayEvent.QueryParams["command"]
	commandOutput := runner.Run(command)
	// Return a message, together with the incoming input...
	return lambhackResponse{
		Message: fmt.Sprintf("Welcome to lambhack!" + command),
		Request: gatewayEvent,
	}, nil
}

func spartaHTMLLambdaFunctions(api *sparta.API) []*sparta.LambdaAWSInfo {
	var lambdaFunctions []*sparta.LambdaAWSInfo
	lambdaFn := sparta.HandleAWSLambda(sparta.LambdaName(lambhack),
		lambhack,
		sparta.IAMRoleDefinition{})

	if nil != api {
		apiGatewayResource, _ := api.NewResource("/lambhack", lambdaFn)

		// We only return http.StatusOK
		apiMethod, apiMethodErr := apiGatewayResource.NewMethod("GET",
			http.StatusOK,
			http.StatusOK)
		if nil != apiMethodErr {
			panic("Failed to create /lambhack resource: " + apiMethodErr.Error())
		}
		// The lambda resource only supports application/json Unmarshallable
		// requests.
		apiMethod.SupportedRequestContentTypes = []string{"application/json"}
	}
	return append(lambdaFunctions, lambdaFn)
}

////////////////////////////////////////////////////////////////////////////////
// Main
func main() {

	// Provision an S3 site
	s3Site, s3SiteErr := sparta.NewS3Site("./resources")
	if s3SiteErr != nil {
		panic("Failed to create S3 Site")
	}

	// Register the function with the API Gateway
	apiStage := sparta.NewStage("prod")
	apiGateway := sparta.NewAPIGateway("lambhack", apiStage)

	// Enable CORS s.t. the S3 site can access the resources
	apiGateway.CORSOptions = &sparta.CORSOptions{
		Headers: map[string]interface{}{
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Origin":  gocf.GetAtt(s3Site.CloudFormationS3ResourceName(), "WebsiteURL"),
		},
	}

	// Deploy it
	stackName := spartaCF.UserScopedStackName("lambhack")
	sparta.Main(stackName,
		fmt.Sprintf("Sacrificial lambs"),
		spartaHTMLLambdaFunctions(apiGateway),
		apiGateway,
		s3Site)
}
