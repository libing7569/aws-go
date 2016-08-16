package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	MIN_CHAN_BUF_SIZE  = 8
	CHAN_BUF_SIZE      = 64
	MAX_ITEMS_PER_PAGE = 1000
)

type S3utils struct {
	svc *s3.S3
}

type Stat struct {
	bucket string
	class  string
	num    int
	size   int64
}

func (s *S3utils) Info(bucket string) []Stat {
	start := time.Now().Unix()
	var wg sync.WaitGroup
	var wg2 sync.WaitGroup
	myinfos := []Stat{}
	total := make(chan Stat, MIN_CHAN_BUF_SIZE)
	glacier := make(chan int64, CHAN_BUF_SIZE)
	standard := make(chan int64, CHAN_BUF_SIZE)
	sia := make(chan int64, CHAN_BUF_SIZE)
	rr := make(chan int64, CHAN_BUF_SIZE)

	params := &s3.ListObjectsInput{
		Bucket:  aws.String(bucket),
		MaxKeys: aws.Int64(MAX_ITEMS_PER_PAGE),
	}

	wg.Add(1)
	go func(param *s3.ListObjectsInput) {
		defer wg.Done()

		err := s.svc.ListObjectsPages(params,
			func(page *s3.ListObjectsOutput, last bool) bool {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for _, object := range page.Contents {
						switch *object.StorageClass {
						case "STANDARD":
							standard <- *object.Size
						case "GLACIER":
							glacier <- *object.Size
						case "STANDARD_IA":
							sia <- *object.Size
						case "REDUCED_REDUNDANCY":
							rr <- *object.Size
						default:
							fmt.Println("error")
						}
					}
				}()
				return true
			},
		)
		if err != nil {
			fmt.Println("Error listing", *params.Bucket, "objects:", err)
		}
	}(params)

	go func() {
		wg.Wait()
		close(standard)
		close(glacier)
		close(sia)
		close(rr)
	}()

	wg2.Add(4)
	go func() {
		defer wg2.Done()
		var sum int64 = 0
		var num int = 0
		for key := range standard {
			sum += key
			num += 1
		}
		total <- Stat{bucket, "STANDARD", num, sum}
	}()

	go func() {
		defer wg2.Done()
		var sum int64 = 0
		var num int = 0
		for key := range glacier {
			sum += key
			num += 1
		}
		total <- Stat{bucket, "GLACIER", num, sum}
	}()

	go func() {
		defer wg2.Done()
		var sum int64 = 0
		var num int = 0
		for key := range sia {
			sum += key
			num += 1
		}
		total <- Stat{bucket, "STANDARD_IA", num, sum}
	}()

	go func() {
		defer wg2.Done()
		var sum int64 = 0
		var num int = 0
		for key := range rr {
			sum += key
			num += 1
		}
		total <- Stat{bucket, "REDUCED_REDUNDANCY", num, sum}
	}()

	go func() {
		wg2.Wait()
		close(total)
	}()

	for t := range total {
		myinfos = append(myinfos, t)
		fmt.Printf("*bucket: %v, class: %v, num: %v, size %v\n", t.bucket, t.class, t.num, t.size)
	}

	end := time.Now().Unix()

	fmt.Printf("Scan Bucket[%v], Cost: %v\n\n", bucket, end-start)
	return myinfos
}

func (s *S3utils) All() {
	start := time.Now().Unix()
	var mwg sync.WaitGroup
	var glacier, standard, sia, rr int64 = 0, 0, 0, 0
	var glaciern, standardn, sian, rrn int = 0, 0, 0, 0
	bs, ok := s.svc.ListBuckets(&s3.ListBucketsInput{})
	ch := make(chan []Stat)
	cnt := 0

	if ok == nil {
		for _, bucket := range bs.Buckets {
			cnt += 1
			mwg.Add(1)
			go func(b string) {
				defer mwg.Done()
				ch <- s.Info(b)
			}(*bucket.Name)
		}

		go func() {
			mwg.Wait()
			close(ch)
		}()

		for qs := range ch {
			for _, q := range qs {
				//fmt.Printf("[%v] class: %v, num: %v, size %v\n", q.Bucket, q.Class, q.Num, q.Size)
				switch q.class {
				case "STANDARD":
					standard += q.size
					standardn += q.num
				case "GLACIER":
					glacier += q.size
					glaciern += q.num
				case "STANDARD_IA":
					sia += q.size
					sian += q.num
				case "REDUCED_REDUNDANCY":
					rr += q.size
					rrn += q.num
				default:
					fmt.Println("Error")
				}
			}
		}
	}
	end := time.Now().Unix()
	fmt.Println("===========TOTAL===========")
	fmt.Printf("standard[%v]: %v\nglacier[%v]: %v\nstandard_ia[%v]: %v\nreduced_redundancy[%v]: %v\n",
		standardn, standard, glaciern, glacier, sian, sia, rrn, rr)
	fmt.Printf("All Cost: %v\n", end-start)
}

func (s *S3utils) Interact() {
	fmt.Println("=========s3=========")

	for {
		fmt.Println("Please input the bucket(none input would triggle an overall scan include all buckets under your aws account):\n")
		reader := bufio.NewReader(os.Stdin)
		data, _, _ := reader.ReadLine()
		bucket := string(data)

		if bucket == "exit" {
			fmt.Println("Bye...")
			break
		} else if bucket == "" {
			s.All()
		} else {
			s.Info(bucket)
		}
	}

}

func main() {
	s3utils := &S3utils{s3.New(session.New(), &aws.Config{Region: aws.String("cn-north-1")})}
	s3utils.Interact()
}
