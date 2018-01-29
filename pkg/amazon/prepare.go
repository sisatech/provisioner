package amazon

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/sisatech/progress"
)

func createInstance(svc *ec2.EC2, availabilityZone *string) (*string, error) {

	// fmt.Printf("createInstance\n")
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
	// fmt.Printf("stopInstance\n")

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
	// fmt.Printf("getVolumeID\n")
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId, nil
}

func getDeviceName(svc *ec2.EC2, instanceID *string) (*string, error) {
	// fmt.Printf("getDeviceName\n")
	di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return nil, err
	}

	return di.Reservations[0].Instances[0].BlockDeviceMappings[0].DeviceName, nil
}

// func getOwnerID(svc *ec2.EC2, instanceID *string) (*string, error) {
func getOwnerID(svc *ec2.EC2) (*string, error) {
	// fmt.Printf("getOwnerID\n")

	// accounts only have access to security groups they own. So the ownerID is unique to them.
	dsgo, err := svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	if len(dsgo.SecurityGroups) > 0 {
		return dsgo.SecurityGroups[0].OwnerId, nil
	}

	// this should never occur as every account should have a default security group.
	return nil, fmt.Errorf("no security groups found")

	// csgo, err := svc.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
	// 	Description: aws.String(""),
	// 	GroupName:   aws.String(""),
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// dsgo, err = svc.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{})
	// if err != nil {
	// 	return nil, err
	// }
	// if len(dsgo.SecurityGroups) > 0 {
	// 	return nil, fmt.Errorf("no security groups found after creating one")
	// }

	// _, err = svc.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
	// 	GroupName: csgo.GroupId,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// return dsgo.SecurityGroups[0].OwnerId, nil

	// di, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
	// 	InstanceIds: aws.StringSlice([]string{*instanceID}),
	// })

	// if err != nil {
	// 	return nil, err
	// }

	// return di.Reservations[0].OwnerId, nil
}

func detachVolume(svc *ec2.EC2, volumeID *string) error {
	// fmt.Printf("detachVolume\n")
	_, err := svc.DetachVolume(&ec2.DetachVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		return err
	}
	return nil
}

func deleteVolume(svc *ec2.EC2, volumeID *string) error {

	err := waitForVolumeToDetach(svc, volumeID)
	if err != nil {
		return err
	}

	// fmt.Printf("deleteVolume\n")
	_, err = svc.DeleteVolume(&ec2.DeleteVolumeInput{
		VolumeId: volumeID,
	})

	if err != nil {
		return err
	}
	return nil
}

func waitForVolumeToDetach(svc *ec2.EC2, volumeID *string) error {
	// fmt.Printf("waitForVolumeToDetach\n")
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func importSnapshot(svc *ec2.EC2, bucket *string, key *string, format *string) (*string, error) {
	// fmt.Printf("importSnapshot\n")
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

func waitUntilSnapshotImported(svc *ec2.EC2, importTaskID *string, pt progress.ProgressTracker) (*string, error) {
	// fmt.Printf("waitUntilSnapshotImported\n")
	initial := pt.Status().Progress
	isImportingSnapshot := true
	var snapshotID *string
	for {
		st, err := svc.DescribeImportSnapshotTasks(&ec2.DescribeImportSnapshotTasksInput{})

		if err != nil {
			return nil, err
		}

		for i := 0; i < len(st.ImportSnapshotTasks); i++ {
			if aws.StringValue(importTaskID) == aws.StringValue(st.ImportSnapshotTasks[i].ImportTaskId) {
				if st.ImportSnapshotTasks[i].SnapshotTaskDetail != nil && st.ImportSnapshotTasks[i].SnapshotTaskDetail.StatusMessage != nil {
					pt.SetStage(fmt.Sprintf("Importing snapshot: %s", *st.ImportSnapshotTasks[i].SnapshotTaskDetail.StatusMessage))
				}
				if st.ImportSnapshotTasks[i].SnapshotTaskDetail != nil && st.ImportSnapshotTasks[i].SnapshotTaskDetail.Progress != nil {
					percentage, _ := strconv.ParseInt(*st.ImportSnapshotTasks[i].SnapshotTaskDetail.Progress, 10, 64)
					pt.SetProgress(float64(percentage) + initial)
				}
				// fmt.Printf("st [%d]: %s\n", i, st.ImportSnapshotTasks[i])
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

	pt.SetProgress(100 + initial)

	return snapshotID, nil
}

func getAvailibityZone(svc *ec2.EC2, region *string) (*string, error) {
	// fmt.Printf("getAvailibityZone\n")
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
	// fmt.Printf("createVolume\n")

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
	// fmt.Printf("deleteSnapshot\n")
	_, err := svc.DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: snapshotID,
	})
	if err != nil {
		return err
	}
	return nil
}

func waitUntilVolumeCreated(svc *ec2.EC2, volumeID *string) error {
	// fmt.Printf("waitUntilVolumeCreated\n")
	err := svc.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{
		VolumeIds: aws.StringSlice([]string{*volumeID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func attachVolume(svc *ec2.EC2, volumeID *string, instanceID *string, deviceName *string) error {
	// fmt.Printf("attachVolume\n")
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

func checkImageExists(svc *ec2.EC2, name string, ownerID *string, overwrite bool) error {
	// fmt.Printf("checkImageExists\n")
	di, _ := svc.DescribeImages(&ec2.DescribeImagesInput{
		Owners: aws.StringSlice([]string{*ownerID}),
	})

	exists := false
	var imageID *string

	for i := 0; i < len(di.Images); i++ {
		if *di.Images[i].Name == name {
			exists = true
			imageID = di.Images[i].ImageId
			break
		}
	}

	if exists {
		if overwrite {
			deleteImage(svc, imageID)
		} else {
			return fmt.Errorf("AMI '%s' already exists", name)
		}
	}

	return nil
}

func createImage(svc *ec2.EC2, instanceID *string, name string, description string) error {
	// fmt.Printf("createImage\n")

	ci, err := svc.CreateImage(&ec2.CreateImageInput{
		InstanceId:  instanceID,
		Name:        aws.String(name),
		Description: aws.String(description),
	})

	if err != nil {
		return err
	}

	imageID := ci.ImageId

	err = svc.WaitUntilImageAvailable(&ec2.DescribeImagesInput{
		ImageIds: aws.StringSlice([]string{*imageID}),
	})

	if err != nil {
		return err
	}

	return nil
}

func deleteImage(svc *ec2.EC2, imageID *string) error {
	// fmt.Printf("deleteImage\n")
	_, err := svc.DeregisterImage(&ec2.DeregisterImageInput{
		ImageId: imageID,
	})
	if err != nil {
		return err
	}

	return nil
}

func deleteInstance(svc *ec2.EC2, instanceID *string) error {
	// fmt.Printf("deleteInstance\n")
	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice([]string{*instanceID}),
	})

	if err != nil {
		return err
	}
	return nil
}

func deleteDisk(p *Provisioner, name string) error {
	// fmt.Printf("deleteDisk\n")
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      p.region,
		Credentials: p.credentials,
	}))
	svc := s3.New(sess)

	_, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: p.bucket,
		Key:    aws.String(name),
	})
	if err != nil {
		return err
	}

	return nil
}

func registerImage(svc *ec2.EC2, snapshotID *string) error {
	svc.RegisterImage(&ec2.RegisterImageInput{
		RootDeviceName: aws.String("/dev/sda1"),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{{
			DeviceName: aws.String("/dev/sda1"),
			Ebs: &ec2.EbsBlockDevice{
				SnapshotId: snapshotID,
			}},
		},
		Name:               aws.String("snap-image1"),
		VirtualizationType: aws.String("hvm"),
	})
	return nil
}

func uploadAndImport(svc *ec2.EC2, p *Provisioner, r io.ReadCloser, name string, c chan *string, pt progress.ProgressTracker) error {
	// fmt.Printf("uploadAndImport\n")
	pt.SetStage("Provisioning.")
	err := p.Provision(name, r)
	if err != nil {
		close(c)
		return err
	}
	pt.IncrementProgress(1)

	pt.SetStage("Importing snapshot.")
	importTaskID, err := importSnapshot(svc, p.bucket, aws.String(name), p.format)
	if err != nil {
		close(c)
		return err
	}

	c <- importTaskID

	return nil
}

// Prepare creates an AMI from a ReadCloser r and names it name
func (p *Provisioner) Prepare(r io.ReadCloser, name, description string, overwriteImage bool, pt progress.ProgressTracker) error {

	pt.Initialize("Provisioning Virtual Machine Image.", 105, progress.UnitStep)

	pt.SetStage("Authenticating with Amazon servers.")
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      p.region,
		Credentials: p.credentials,
	}))
	svc := ec2.New(sess)
	pt.IncrementProgress(1)

	// c := make(chan *string)
	// go uploadAndImport(svc, p, r, name, c, pt)

	pt.SetStage("Requesting owner ID.")
	ownerID, err := getOwnerID(svc)
	if err != nil {
		return err
	}
	pt.IncrementProgress(1)

	pt.SetStage("Cheching if AMI already exists.")
	err = checkImageExists(svc, name, ownerID, overwriteImage)
	if err != nil {
		return err
	}
	pt.IncrementProgress(1)

	pt.SetStage("Provisioning.")
	err = p.Provision(name, r)
	if err != nil {
		return err
	}
	pt.IncrementProgress(1)

	pt.SetStage("Importing snapshot.")
	importTaskID, err := importSnapshot(svc, p.bucket, aws.String(name), p.format)
	if err != nil {
		return err
	}
	defer deleteDisk(p, name)

	// pt.SetStage("Requesting list of availability zones.")
	// availabilityZone, err := getAvailibityZone(svc, p.region)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Creating generic VM instance.")
	// instanceID, err := createInstance(svc, availabilityZone)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// defer deleteInstance(svc, instanceID)

	// pt.SetStage("Stopping generic VM instance.")
	// err = stopInstance(svc, instanceID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Requesting volume ID.")
	// volumeID, err := getVolumeID(svc, instanceID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Requesting device name")
	// deviceName, err := getDeviceName(svc, instanceID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Requesting owner ID.")
	// ownerID, err := getOwnerID(svc, instanceID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Cheching if AMI already exists.")
	// err = checkImageExists(svc, name, ownerID, overwriteImage)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Detaching volume.")
	// err = detachVolume(svc, volumeID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Deleting volume.")
	// err = deleteVolume(svc, volumeID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// importTaskID, ok := <-c
	// if ok == false {
	// 	return fmt.Errorf("Uploading or importing failed")
	// }
	snapshotID, err := waitUntilSnapshotImported(svc, importTaskID, pt)
	if err != nil {
		return err
	}

	// defer deleteSnapshot(svc, snapshotID)

	// err = registerImage(svc, snapshotID)
	// if err != nil {
	// 	return err
	// }

	// pt.SetStage("Creating volume.")
	// volumeID, err = createVolume(svc, availabilityZone, snapshotID)
	// if err != nil {
	// 	return err
	// }

	// err = waitUntilVolumeCreated(svc, volumeID)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Attaching volume.")
	// err = attachVolume(svc, volumeID, instanceID, deviceName)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	// pt.SetStage("Creating image.")
	// err = createImage(svc, instanceID, name, description)
	// if err != nil {
	// 	return err
	// }
	// pt.IncrementProgress(1)

	pt.SetStage("Regiestering image.")
	err = registerImage(svc, snapshotID)
	if err != nil {
		return err
	}
	pt.IncrementProgress(1)

	return nil
}
