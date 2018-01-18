package amazon

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func createInstance(svc *ec2.EC2) *string {

	ri, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:          aws.String("ami-7dce6507"),
		InstanceType:     aws.String("t2.micro"),
		KeyName:          aws.String("testdetach"),
		SecurityGroupIds: aws.StringSlice([]string{"sg-63bc3916"}),
		MaxCount:         aws.Int64(1),
		MinCount:         aws.Int64(1),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("run-instances:\n%s\n", ri)

	instanceID := ri.Instances[0].InstanceId

	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance [%s] now running.\n", *instanceID)

	return instanceID
}

func stopInstance(svc *ec2.EC2, instanceID *string) {
	si, err := svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("stop-instances:\n%s\n", si)

	err = svc.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Instance [%s] now stopped.\n", *instanceID)
}

func getVolumeID(svc *ec2.EC2, instanceID *string) *string {
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("describe-instances:\n%s\n", di)

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId
}

func getDeviceName(svc *ec2.EC2, instanceID *string) *string {
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("describe-instances:\n%s\n", di)

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].DeviceName
}

func detachVolume(svc *ec2.EC2, volumeID *string) {
	dv, err := svc.DetachVolume(&ec2.DetachVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("detach-volume:\n%s\n", dv)
}

func deleteVolume(svc *ec2.EC2, volumeID *string) {
	del, err := svc.DeleteVolume(&ec2.DeleteVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("delete-volume:\n%s\n", del)
}

func waitForVolumeToDetach(svc *ec2.EC2, volumeID *string) {
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Volume [%s] now detached.\n", *volumeID)
}

func importSnapshot(svc *ec2.EC2, bucket *string, key *string) *string {
	is, err := svc.ImportSnapshot(&ec2.ImportSnapshotInput{
		Description: aws.String("temp snapshot for " + *key),
		DiskContainer: &ec2.SnapshotDiskContainer{
			Description: aws.String("temp snapshot for " + *key),
			Format:      aws.String("raw"),
			UserBucket: &ec2.UserBucket{
				S3Bucket: bucket,
				S3Key:    key,
			},
		},
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("import-snapshot:\n%s\n", is)

	return is.ImportTaskId
}

func waitUntilSnapshotImported(svc *ec2.EC2, importTaskID *string) *string {

	isImportingSnapshot := true
	var snapshotID *string
	for {
		st, err := svc.DescribeImportSnapshotTasks(&ec2.DescribeImportSnapshotTasksInput{})

		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("DescribeImportSnapshotTasks:\n%s\n", st)

		for i := 0; i < len(st.ImportSnapshotTasks); i++ {
			fmt.Printf("st [%d]: %s\n", i, st.ImportSnapshotTasks[i])
			if aws.StringValue(importTaskID) == aws.StringValue(st.ImportSnapshotTasks[i].ImportTaskId) {
				if aws.StringValue(st.ImportSnapshotTasks[i].SnapshotTaskDetail.Status) == "completed" {
					snapshotID = st.ImportSnapshotTasks[i].SnapshotTaskDetail.SnapshotId
					isImportingSnapshot = false
				}
				break
			}
		}

		if isImportingSnapshot != true {
			break
		}
		time.Sleep(15 * time.Second)

	}

	return snapshotID
}

func createVolume(svc *ec2.EC2, availabilityZone *string, snapshotID *string) *string {
	cv, err := svc.CreateVolume(&ec2.CreateVolumeInput{
		AvailabilityZone: availabilityZone,
		SnapshotId:       snapshotID,
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("create-volume:\n%s\n", cv)

	return cv.VolumeId
}

func deleteSnapshot(svc *ec2.EC2, snapshotID *string) {
	ds, err := svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("delete-snapshot:\n%s\n", ds)
}

func waitUntilVolumeCreated(svc *ec2.EC2, volumeID *string) {
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Volume [%s] now available.\n", *volumeID)
}

func attachVolume(svc *ec2.EC2, volumeID *string, instanceID *string, deviceName *string) {
	av, err := svc.AttachVolume(&ec2.AttachVolumeInput{
		VolumeId:   volumeID,
		InstanceId: instanceID,
		Device:     deviceName,
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("attach-volume:\n%s\n", av)
}

func createImage(svc *ec2.EC2, instanceID *string, name string) {
	ci, err := svc.CreateImage(&ec2.CreateImageInput{
		InstanceId: instanceID,
		Name:       aws.String(name),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("attach-volume:\n%s\n", ci)
}

func deleteInstance(svc *ec2.EC2, instanceID *string) {
	ti, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("terminate-instances:\n%s\n", ti)
}

// Prepare ...
func Prepare(p *Provisioner, f string, r io.ReadCloser, name string) {

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      p.region,
		Credentials: p.credentials,
	}))
	svc := ec2.New(sess)

	instanceID := createInstance(svc)
	stopInstance(svc, instanceID)
	volumeID := getVolumeID(svc, instanceID)
	deviceName := getDeviceName(svc, instanceID)
	detachVolume(svc, volumeID)
	waitForVolumeToDetach(svc, volumeID)
	deleteVolume(svc, volumeID)

	p.Provision(f, r)

	importTaskID := importSnapshot(svc, p.bucket, aws.String(f))
	snapshotID := waitUntilSnapshotImported(svc, importTaskID)

	volumeID = createVolume(svc, aws.String("us-east-1c"), snapshotID)

	deleteSnapshot(svc, snapshotID)
	waitUntilVolumeCreated(svc, volumeID)
	attachVolume(svc, volumeID, instanceID, deviceName)
	createImage(svc, instanceID, name)
	deleteInstance(svc, instanceID)
	// spawn from ami

}
