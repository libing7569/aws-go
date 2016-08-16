package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
)

type TagsUtils map[string]string

func New(t ...string) TagsUtils {
	tmp := make(map[string]string)
	if len(t) < 0 || len(t)%2 != 0 {
		return nil
	}

	for i := 0; i < len(t); i += 2 {
		tmp[t[i]] = t[i+1]
	}

	return tmp
}

func NewFromEc2Resource(id string) TagsUtils {
	return getTagsEc2(id)
}

func NewFromS3(bucket string) TagsUtils {
	return getTagsS3(bucket)
}

func getTagsEc2(id string) map[string]string {
	tmp := make(map[string]string)
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})
	filters := []*ec2.Filter{&ec2.Filter{
		Name:   aws.String("resource-id"),
		Values: []*string{aws.String(id)},
	}}
	res, err := svc.DescribeTags(&ec2.DescribeTagsInput{
		Filters: filters,
	})
	if err == nil {
		for _, t := range res.Tags {
			tmp[*t.Key] = *t.Value
		}
	} else {
		fmt.Println(err)
		tmp = nil
	}

	return tmp
}

func getTagsS3(bucket string) map[string]string {
	tmp := make(map[string]string)
	svc := s3.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})
	res, _ := svc.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: aws.String(bucket)})

	for _, tag := range res.TagSet {
		tmp[*tag.Key] = *tag.Value
	}

	return tmp
}

func (tu TagsUtils) descTags() {
	for k, v := range tu {
		fmt.Println(k, ": ", v)
	}
	fmt.Println("")
}

func (tu TagsUtils) tagEc2Resouces(rids []string) {
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})
	tags := []*ec2.Tag{}
	for k, v := range tu {
		tags = append(tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	ids := []*string{}

	for _, id := range rids {
		ids = append(ids, aws.String(id))
	}

	_, errtag := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: ids,
		Tags:      tags,
	})

	if errtag != nil {
		log.Println("Could not create tags for instance", errtag)
		return
	}

	log.Println("Successfully tagged instance")
}

func (tu TagsUtils) tagEc2ByFilters(conds map[string][]*string) {
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})
	filters := []*ec2.Filter{}
	for k, v := range conds {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String(k),
			Values: v,
		})
	}

	params := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	resp, err := svc.DescribeInstances(params)
	if err != nil {
		panic(err)
	}

	ids := []string{}
	// resp has all of the response data, pull out instance IDs:
	fmt.Println("> Number of reservation sets: ", len(resp.Reservations))
	for idx, res := range resp.Reservations {
		fmt.Println("  > Number of instances: ", len(res.Instances))
		for _, inst := range resp.Reservations[idx].Instances {
			fmt.Println("    - Instance ID: ", *inst.InstanceId)
			ids = append(ids, *inst.InstanceId)
		}
	}

	tu.tagEc2Resouces(ids)
}

func (tu TagsUtils) tagS3Buckets(bs []string) {
	for _, b := range bs {
		tu.tagS3Bucket(b)
	}
}

func (tu TagsUtils) tagS3Bucket(bucket string) {
	svc := s3.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})

	tags := []*s3.Tag{}
	for k, v := range tu {
		tags = append(tags, &s3.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	res, err := svc.PutBucketTagging(&s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &s3.Tagging{
			TagSet: tags,
		},
	})

	if err != nil {
		log.Println("Could not create tags for this bucket", res, err)
		return
	}
	log.Println("Successfully tagged bucket")
	//fmt.Println(res, err)
}

func getInputString() string {
	reader := bufio.NewReader(os.Stdin)
	data, _, _ := reader.ReadLine()
	return string(data)
}

func main() {
	srcType := flag.String("srctype", "ec2", "Input src type, available values include ec2, s3")
	dstType := flag.String("dsttype", "ec2", "Input dst type, available values include ec2, s3")
	flag.Parse()
	//fmt.Println(*srcType, *dstType)
	srid, sbucket, drid, dbucket := "", "", "", ""
	var tagsUtils TagsUtils = nil

	switch *srcType {
	case "ec2":
		fmt.Println("Please input the source resouce id:")
		srid = getInputString()
		tagsUtils = NewFromEc2Resource(srid)
	case "s3":
		fmt.Println("Please input the source bucket name:")
		sbucket = getInputString()
		tagsUtils = NewFromS3(sbucket)
	case "manual":
		fmt.Println("Hello")
	default:
		log.Fatal("No such type")
	}

chg:
	for {
		fmt.Println("-----Current tags-----")
		tagsUtils.descTags()
		fmt.Println("edit(a|c|d|q)?")
		nChg := getInputString()
		switch nChg {
		case "a":
			fallthrough
		case "c":
			fmt.Println("Please input key:")
			key := getInputString()
			fmt.Println("Please input value:")
			value := getInputString()
			tagsUtils[key] = value
		case "d":
			fmt.Println("Please input key:")
			key := getInputString()
			delete(tagsUtils, key)
		case "q":
			break chg
		default:
		}
	}

	switch *dstType {
	case "ec2":
		fmt.Println("Please input the dest resouce id('id1,id2,id3,...'):")
		drid = getInputString()
		drids := strings.Split(drid, ",")
		tagsUtils.tagEc2Resouces(drids)
	case "s3":
		fmt.Println("Please input the dest bucket name('bucket1,bucket2,bucket3,...'):")
		dbucket = getInputString()
		dbuckets := strings.Split(dbucket, ",")
		tagsUtils.tagS3Buckets(dbuckets)
	default:
		log.Fatal("No such type")
	}
}
