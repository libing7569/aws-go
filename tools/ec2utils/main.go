package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})

	instanceIDsToStop := aws.StringSlice([]string{"i-efd9f6d7"})
	fmt.Println("Instances are stopped start.")
	//_, _ = svc.StopInstances(&ec2.StopInstancesInput{
	//InstanceIds: instanceIDsToStop,
	//})
	//svc.StopInstances(describeInstancesInput)
	describeInstancesInput := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDsToStop,
	}

	if err := svc.WaitUntilInstanceStopped(describeInstancesInput); err != nil {
		panic(err)
	}

	fmt.Println("Instances are stopped end.")
}
