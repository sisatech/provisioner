package amazon

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func createInstance(svc *ec2.EC2, availabilityZone *string) (*string, error) {

	fmt.Printf("createInstance\n")
	ri, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String("ami-7dce6507"),
		InstanceType: aws.String("t2.micro"),
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		Placement: &ec2.Placement{
			AvailabilityZone: availabilityZone,
		},
	})

	if err != nil {
		return nil, err
	}

	instanceID := ri.Instances[0].InstanceId

	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return instanceID, nil
}

func stopInstance(svc *ec2.EC2, instanceID *string) error {

	fmt.Printf("stopInstance\n")

	_, err := svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return err
	}

	err = svc.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func getVolumeID(svc *ec2.EC2, instanceID *string) (*string, error) {
	fmt.Printf("getVolumeID\n")
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId, nil
}

func getDeviceName(svc *ec2.EC2, instanceID *string) (*string, error) {
	fmt.Printf("getDeviceName\n")
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].DeviceName, nil
}

func detachVolume(svc *ec2.EC2, volumeID *string) error {
	fmt.Printf("detachVolume\n")
	_, err := svc.DetachVolume(&ec2.DetachVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		return err
	}
	return nil
}

func deleteVolume(svc *ec2.EC2, volumeID *string) error {
	fmt.Printf("deleteVolume\n")
	_, err := svc.DeleteVolume(&ec2.DeleteVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		return err
	}
	return nil
}

func waitForVolumeToDetach(svc *ec2.EC2, volumeID *string) error {
	fmt.Printf("waitForVolumeToDetach\n")
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func importSnapshot(svc *ec2.EC2, bucket *string, key *string, format *string) (*string, error) {
	fmt.Printf("importSnapshot\n")
	is, err := svc.ImportSnapshot(&ec2.ImportSnapshotInput{
		Description: aws.String("temp snapshot for " + *key),
		DiskContainer: &ec2.SnapshotDiskContainer{
			Description: aws.String("temp snapshot for " + *key),
			Format:      format,
			UserBucket: &ec2.UserBucket{
				S3Bucket: bucket,
				S3Key:    key,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	return is.ImportTaskId, nil
}

func waitUntilSnapshotImported(svc *ec2.EC2, importTaskID *string) (*string, error) {
	fmt.Printf("waitUntilSnapshotImported\n")
	isImportingSnapshot := true
	var snapshotID *string
	for {
		st, err := svc.DescribeImportSnapshotTasks(&ec2.DescribeImportSnapshotTasksInput{})

		if err != nil {
			return nil, err
		}

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

	return snapshotID, nil
}

func getAvailibityZone(svc *ec2.EC2, region *string) (*string, error) {
	fmt.Printf("getAvailibityZone\n")
	zones, err := svc.DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})

	if err != nil {
		return nil, err
	}

	var availabilityZone *string

	for i := 0; i < len(zones.AvailabilityZones); i++ {
		if *zones.AvailabilityZones[i].RegionName == *region {
			availabilityZone = zones.AvailabilityZones[i].ZoneName
			break
		}
	}

	return availabilityZone, nil
}

func createVolume(svc *ec2.EC2, availabilityZone *string, snapshotID *string) (*string, error) {
	fmt.Printf("createVolume\n")

	cv, err := svc.CreateVolume(&ec2.CreateVolumeInput{
		AvailabilityZone: availabilityZone,
		SnapshotId:       snapshotID,
	})

	if err != nil {
		return nil, err
	}

	return cv.VolumeId, nil
}

func deleteSnapshot(svc *ec2.EC2, snapshotID *string) error {
	fmt.Printf("deleteSnapshot\n")
	_, err := svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})
	if err != nil {
		return err
	}
	return nil
}

func waitUntilVolumeCreated(svc *ec2.EC2, volumeID *string) error {
	fmt.Printf("waitUntilVolumeCreated\n")
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func attachVolume(svc *ec2.EC2, volumeID *string, instanceID *string, deviceName *string) error {
	fmt.Printf("attachVolume\n")
	_, err := svc.AttachVolume(&ec2.AttachVolumeInput{
		VolumeId:   volumeID,
		InstanceId: instanceID,
		Device:     deviceName,
	})

	if err != nil {
		return err
	}
	return nil
}

func createImage(svc *ec2.EC2, instanceID *string, name string) error {
	fmt.Printf("createImage\n")
	_, err := svc.CreateImage(&ec2.CreateImageInput{
		InstanceId: instanceID,
		Name:       aws.String(name),
	})

	if err != nil {
		return err
	}
	return nil
}

func deleteInstance(svc *ec2.EC2, instanceID *string) error {
	fmt.Printf("deleteInstance\n")
	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return err
	}
	return nil
}

// Prepare ...
func (p *Provisioner) Prepare(r io.ReadCloser, name string) error {

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      p.region,
		Credentials: p.credentials,
	}))
	svc := ec2.New(sess)

	availabilityZone, err := getAvailibityZone(svc, p.region)
	if err != nil {
		return err
	}

	instanceID, err := createInstance(svc, availabilityZone)
	if err != nil {
		return err
	}

	err = stopInstance(svc, instanceID)
	if err != nil {
		return err
	}

	volumeID, err := getVolumeID(svc, instanceID)
	if err != nil {
		return err
	}

	deviceName, err := getDeviceName(svc, instanceID)
	if err != nil {
		return err
	}

	err = detachVolume(svc, volumeID)
	if err != nil {
		return err
	}

	err = waitForVolumeToDetach(svc, volumeID)
	if err != nil {
		return err
	}

	err = deleteVolume(svc, volumeID)
	if err != nil {
		return err
	}

	fmt.Printf("UploadingDisk\n")
	err = p.Provision(name, r)
	if err != nil {
		return err
	}

	importTaskID, err := importSnapshot(svc, p.bucket, aws.String(name), p.format)
	if err != nil {
		return err
	}
	snapshotID, err := waitUntilSnapshotImported(svc, importTaskID)
	if err != nil {
		return err
	}

	volumeID, err = createVolume(svc, availabilityZone, snapshotID)
	if err != nil {
		return err
	}

	err = deleteSnapshot(svc, snapshotID)
	if err != nil {
		return err
	}
	err = waitUntilVolumeCreated(svc, volumeID)
	if err != nil {
		return err
	}
	err = attachVolume(svc, volumeID, instanceID, deviceName)
	if err != nil {
		return err
	}
	err = createImage(svc, instanceID, name)
	if err != nil {
		return err
	}
	err = deleteInstance(svc, instanceID)
	if err != nil {
		return err
	}

	// check if ami exists
	// spawn from ami

	return nil
}
