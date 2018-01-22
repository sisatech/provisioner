package amazon

import (
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func createInstance(svc *ec2.EC2) (*string, error) {

	ri, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String("ami-7dce6507"),
		InstanceType: aws.String("t2.micro"),
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
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
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId, nil
}

func getDeviceName(svc *ec2.EC2, instanceID *string) (*string, error) {
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].DeviceName, nil
}

func detachVolume(svc *ec2.EC2, volumeID *string) error {
	_, err := svc.DetachVolume(&ec2.DetachVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		return err
	}
	return nil
}

func deleteVolume(svc *ec2.EC2, volumeID *string) error {
	_, err := svc.DeleteVolume(&ec2.DeleteVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		return err
	}
	return nil
}

func waitForVolumeToDetach(svc *ec2.EC2, volumeID *string) error {
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func importSnapshot(svc *ec2.EC2, bucket *string, key *string, format *string) (*string, error) {
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

	isImportingSnapshot := true
	var snapshotID *string
	for {
		st, err := svc.DescribeImportSnapshotTasks(&ec2.DescribeImportSnapshotTasksInput{})

		if err != nil {
			return nil, err
		}

		for i := 0; i < len(st.ImportSnapshotTasks); i++ {
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

func createVolume(svc *ec2.EC2, region *string, snapshotID *string) (*string, error) {

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
	_, err := svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})
	if err != nil {
		return err
	}
	return nil
}

func waitUntilVolumeCreated(svc *ec2.EC2, volumeID *string) error {
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func attachVolume(svc *ec2.EC2, volumeID *string, instanceID *string, deviceName *string) error {
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
	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return err
	}
	return nil
}

// Prepare ...
func (p *Provisioner) Prepare(f string, r io.ReadCloser, name string) error {

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      p.region,
		Credentials: p.credentials,
	}))
	svc := ec2.New(sess)

	instanceID, err := createInstance(svc)
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

	err = p.Provision(f, r)
	if err != nil {
		return err
	}

	importTaskID, err := importSnapshot(svc, p.bucket, aws.String(f), p.format)
	if err != nil {
		return err
	}
	snapshotID, err := waitUntilSnapshotImported(svc, importTaskID)
	if err != nil {
		return err
	}

	volumeID, err = createVolume(svc, p.region, snapshotID)
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

	err = deleteVolume(svc, volumeID)
	if err != nil {
		return err
	}
	// spawn from ami

	return nil
}
