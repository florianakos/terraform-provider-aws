// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ecs

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKDataSource("aws_ecs_task_execution")
func DataSourceTaskExecution() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataSourceTaskExecutionRead,

		Schema: map[string]*schema.Schema{
			"capacity_provider_strategy": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"base": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(0, 100000),
						},
						"capacity_provider": {
							Type:     schema.TypeString,
							Required: true,
						},
						"weight": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(0, 1000),
						},
					},
				},
			},
			"client_token": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"cluster": {
				Type:     schema.TypeString,
				Required: true,
			},
			"desired_count": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 10),
			},
			"enable_ecs_managed_tags": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"enable_execute_command": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"group": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"launch_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(ecs.LaunchType_Values(), false),
			},
			names.AttrNetworkConfiguration: {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						names.AttrSecurityGroups: {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"assign_public_ip": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"overrides": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"container_overrides": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"command": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"cpu": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"environment": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrKey: {
													Type:     schema.TypeString,
													Required: true,
												},
												names.AttrValue: {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
									"memory": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"memory_reservation": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									names.AttrName: {
										Type:     schema.TypeString,
										Required: true,
									},
									"resource_requirements": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrType: {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: validation.StringInSlice(ecs.ResourceType_Values(), false),
												},
												names.AttrValue: {
													Type:     schema.TypeString,
													Required: true,
												},
											},
										},
									},
								},
							},
						},
						"cpu": {
							Type:     schema.TypeString,
							Optional: true,
						},
						names.AttrExecutionRoleARN: {
							Type:     schema.TypeString,
							Optional: true,
						},
						"inference_accelerator_overrides": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									names.AttrDeviceName: {
										Type:     schema.TypeString,
										Optional: true,
									},
									"device_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"memory": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"task_role_arn": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"placement_constraints": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 10,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						names.AttrExpression: {
							Type:     schema.TypeString,
							Optional: true,
						},
						names.AttrType: {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice(ecs.PlacementConstraintType_Values(), false),
						},
					},
				},
			},
			"placement_strategy": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 5,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"field": {
							Type:     schema.TypeString,
							Optional: true,
						},
						names.AttrType: {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"platform_version": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"propagate_tags": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(ecs.PropagateTags_Values(), false),
			},
			"reference_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"started_by": {
				Type:     schema.TypeString,
				Optional: true,
			},
			names.AttrTags: tftags.TagsSchema(),
			"task_arns": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"task_definition": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

const (
	DSNameTaskExecution = "Task Execution Data Source"
)

func dataSourceTaskExecutionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).ECSConn(ctx)

	cluster := d.Get("cluster").(string)
	taskDefinition := d.Get("task_definition").(string)
	d.SetId(strings.Join([]string{cluster, taskDefinition}, ","))

	input := ecs.RunTaskInput{
		Cluster:        aws.String(cluster),
		TaskDefinition: aws.String(taskDefinition),
	}

	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(ctx, d.Get(names.AttrTags).(map[string]interface{})))
	if len(tags) > 0 {
		input.Tags = Tags(tags.IgnoreAWS())
	}

	if v, ok := d.GetOk("capacity_provider_strategy"); ok {
		input.CapacityProviderStrategy = expandCapacityProviderStrategy(v.(*schema.Set))
	}
	if v, ok := d.GetOk("client_token"); ok {
		input.ClientToken = aws.String(v.(string))
	}
	if v, ok := d.GetOk("desired_count"); ok {
		input.Count = aws.Int64(int64(v.(int)))
	}
	if v, ok := d.GetOk("enable_ecs_managed_tags"); ok {
		input.EnableECSManagedTags = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("enable_execute_command"); ok {
		input.EnableExecuteCommand = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("group"); ok {
		input.Group = aws.String(v.(string))
	}
	if v, ok := d.GetOk("launch_type"); ok {
		input.LaunchType = aws.String(v.(string))
	}
	if v, ok := d.GetOk(names.AttrNetworkConfiguration); ok {
		input.NetworkConfiguration = expandNetworkConfiguration(v.([]interface{}))
	}
	if v, ok := d.GetOk("overrides"); ok {
		input.Overrides = expandTaskOverride(v.([]interface{}))
	}
	if v, ok := d.GetOk("placement_constraints"); ok {
		pc, err := expandPlacementConstraints(v.(*schema.Set).List())
		if err != nil {
			return create.AppendDiagError(diags, names.ECS, create.ErrActionCreating, DSNameTaskExecution, d.Id(), err)
		}
		input.PlacementConstraints = pc
	}
	if v, ok := d.GetOk("placement_strategy"); ok {
		ps, err := expandPlacementStrategy(v.([]interface{}))
		if err != nil {
			return create.AppendDiagError(diags, names.ECS, create.ErrActionCreating, DSNameTaskExecution, d.Id(), err)
		}
		input.PlacementStrategy = ps
	}
	if v, ok := d.GetOk("platform_version"); ok {
		input.PlatformVersion = aws.String(v.(string))
	}
	if v, ok := d.GetOk("propagate_tags"); ok {
		input.PropagateTags = aws.String(v.(string))
	}
	if v, ok := d.GetOk("reference_id"); ok {
		input.ReferenceId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("started_by"); ok {
		input.StartedBy = aws.String(v.(string))
	}

	out, err := conn.RunTaskWithContext(ctx, &input)
	if err != nil {
		return create.AppendDiagError(diags, names.ECS, create.ErrActionCreating, DSNameTaskExecution, d.Id(), err)
	}
	if out == nil || len(out.Tasks) == 0 {
		return create.AppendDiagError(diags, names.ECS, create.ErrActionCreating, DSNameTaskExecution, d.Id(), tfresource.NewEmptyResultError(input))
	}

	var taskArns []*string
	for _, t := range out.Tasks {
		taskArns = append(taskArns, t.TaskArn)
	}
	d.Set("task_arns", flex.FlattenStringList(taskArns))

	return diags
}

func expandTaskOverride(tfList []interface{}) *ecs.TaskOverride {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &ecs.TaskOverride{}
	tfMap := tfList[0].(map[string]interface{})

	if v, ok := tfMap["cpu"]; ok {
		apiObject.Cpu = aws.String(v.(string))
	}
	if v, ok := tfMap["memory"]; ok {
		apiObject.Memory = aws.String(v.(string))
	}
	if v, ok := tfMap[names.AttrExecutionRoleARN]; ok {
		apiObject.ExecutionRoleArn = aws.String(v.(string))
	}
	if v, ok := tfMap["task_role_arn"]; ok {
		apiObject.TaskRoleArn = aws.String(v.(string))
	}
	if v, ok := tfMap["inference_accelerator_overrides"]; ok {
		apiObject.InferenceAcceleratorOverrides = expandInferenceAcceleratorOverrides(v.(*schema.Set))
	}
	if v, ok := tfMap["container_overrides"]; ok {
		apiObject.ContainerOverrides = expandContainerOverride(v.([]interface{}))
	}

	return apiObject
}

func expandInferenceAcceleratorOverrides(tfSet *schema.Set) []*ecs.InferenceAcceleratorOverride {
	if tfSet.Len() == 0 {
		return nil
	}
	apiObject := make([]*ecs.InferenceAcceleratorOverride, 0)

	for _, item := range tfSet.List() {
		tfMap := item.(map[string]interface{})
		iao := &ecs.InferenceAcceleratorOverride{
			DeviceName: aws.String(tfMap[names.AttrDeviceName].(string)),
			DeviceType: aws.String(tfMap["device_type"].(string)),
		}
		apiObject = append(apiObject, iao)
	}

	return apiObject
}

func expandContainerOverride(tfList []interface{}) []*ecs.ContainerOverride {
	if len(tfList) == 0 {
		return nil
	}
	apiObject := make([]*ecs.ContainerOverride, 0)

	for _, item := range tfList {
		tfMap := item.(map[string]interface{})
		co := &ecs.ContainerOverride{
			Name: aws.String(tfMap[names.AttrName].(string)),
		}
		if v, ok := tfMap["command"]; ok {
			commandStrings := v.([]interface{})
			co.Command = flex.ExpandStringList(commandStrings)
		}
		if v, ok := tfMap["cpu"]; ok {
			co.Cpu = aws.Int64(int64(v.(int)))
		}
		if v, ok := tfMap["environment"]; ok {
			co.Environment = expandTaskEnvironment(v.(*schema.Set))
		}
		if v, ok := tfMap["memory"]; ok {
			co.Memory = aws.Int64(int64(v.(int)))
		}
		if v, ok := tfMap["memory_reservation"]; ok {
			co.MemoryReservation = aws.Int64(int64(v.(int)))
		}
		if v, ok := tfMap["resource_requirements"]; ok {
			co.ResourceRequirements = expandResourceRequirements(v.(*schema.Set))
		}
		apiObject = append(apiObject, co)
	}

	return apiObject
}

func expandTaskEnvironment(tfSet *schema.Set) []*ecs.KeyValuePair {
	if tfSet.Len() == 0 {
		return nil
	}
	apiObject := make([]*ecs.KeyValuePair, 0)

	for _, item := range tfSet.List() {
		tfMap := item.(map[string]interface{})
		te := &ecs.KeyValuePair{
			Name:  aws.String(tfMap[names.AttrKey].(string)),
			Value: aws.String(tfMap[names.AttrValue].(string)),
		}
		apiObject = append(apiObject, te)
	}

	return apiObject
}

func expandResourceRequirements(tfSet *schema.Set) []*ecs.ResourceRequirement {
	if tfSet.Len() == 0 {
		return nil
	}

	apiObject := make([]*ecs.ResourceRequirement, 0)
	for _, item := range tfSet.List() {
		tfMap := item.(map[string]interface{})
		rr := &ecs.ResourceRequirement{
			Type:  aws.String(tfMap[names.AttrType].(string)),
			Value: aws.String(tfMap[names.AttrValue].(string)),
		}
		apiObject = append(apiObject, rr)
	}

	return apiObject
}
