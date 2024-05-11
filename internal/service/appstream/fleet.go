// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package appstream

import (
	"context"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/appstream"
	"github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_appstream_fleet", name="Fleet")
// @Tags(identifierAttribute="arn")
func ResourceFleet() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceFleetCreate,
		ReadWithoutTimeout:   resourceFleetRead,
		UpdateWithoutTimeout: resourceFleetUpdate,
		DeleteWithoutTimeout: resourceFleetDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: customdiff.Sequence(
			resourceFleetCustDiff,
			verify.SetTagsDiff,
		),

		Schema: map[string]*schema.Schema{
			names.AttrARN: {
				Type:     schema.TypeString,
				Computed: true,
			},
			"compute_capacity": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"available": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"desired_instances": {
							Type:     schema.TypeInt,
							Optional: true,
							ExactlyOneOf: []string{
								"compute_capacity.0.desired_sessions",
							},
						},
						"desired_sessions": {
							Type:     schema.TypeInt,
							Optional: true,
							ExactlyOneOf: []string{
								"compute_capacity.0.desired_instances",
							},
						},
						"in_use": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"running": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"created_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			names.AttrDescription: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			"disconnect_timeout_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(60, 360000),
			},
			names.AttrDisplayName: {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringLenBetween(0, 100),
			},
			"domain_join_info": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"directory_name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"organizational_unit_distinguished_name": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
			"enable_default_internet_access": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"fleet_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(appstream.FleetType_Values(), false),
			},
			"iam_role_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: verify.ValidARN,
			},
			"idle_disconnect_timeout_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      0,
				ValidateFunc: validation.IntBetween(60, 360000),
			},
			"image_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"image_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			names.AttrInstanceType: {
				Type:     schema.TypeString,
				Required: true,
			},
			"max_sessions_per_instance": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"max_user_duration_in_seconds": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.IntBetween(600, 432000),
			},
			names.AttrName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"stream_view": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validation.StringInSlice(appstream.StreamView_Values(), false),
			},
			names.AttrState: {
				Type:     schema.TypeString,
				Computed: true,
			},
			names.AttrVPCConfig: {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						names.AttrSecurityGroupIDs: {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						names.AttrSubnetIDs: {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
		},
	}
}

func resourceFleetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppStreamConn(ctx)
	input := &appstream.CreateFleetInput{
		Name:            aws.String(d.Get(names.AttrName).(string)),
		InstanceType:    aws.String(d.Get(names.AttrInstanceType).(string)),
		ComputeCapacity: expandComputeCapacity(d.Get("compute_capacity").([]interface{})),
		Tags:            getTagsIn(ctx),
	}

	if v, ok := d.GetOk(names.AttrDescription); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("disconnect_timeout_in_seconds"); ok {
		input.DisconnectTimeoutInSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("idle_disconnect_timeout_in_seconds"); ok {
		input.IdleDisconnectTimeoutInSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk(names.AttrDisplayName); ok {
		input.DisplayName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("domain_join_info"); ok {
		input.DomainJoinInfo = expandDomainJoinInfo(v.([]interface{}))
	}

	if v, ok := d.GetOk("enable_default_internet_access"); ok {
		input.EnableDefaultInternetAccess = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("fleet_type"); ok {
		input.FleetType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("image_name"); ok {
		input.ImageName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("image_arn"); ok {
		input.ImageArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("iam_role_arn"); ok {
		input.IamRoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("max_sessions_per_instance"); ok {
		input.MaxSessionsPerInstance = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("max_user_duration_in_seconds"); ok {
		input.MaxUserDurationInSeconds = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("stream_view"); ok {
		input.StreamView = aws.String(v.(string))
	}

	if v, ok := d.GetOk(names.AttrVPCConfig); ok {
		input.VpcConfig = expandVPCConfig(v.([]interface{}))
	}

	var err error
	var output *appstream.CreateFleetOutput
	err = retry.RetryContext(ctx, fleetOperationTimeout, func() *retry.RetryError {
		output, err = conn.CreateFleetWithContext(ctx, input)
		if err != nil {
			if tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
				return retry.RetryableError(err)
			}

			if tfawserr.ErrCodeEquals(err, appstream.ErrCodeConcurrentModificationException) {
				return retry.RetryableError(err)
			}

			// Retry for IAM eventual consistency on error:
			if tfawserr.ErrMessageContains(err, appstream.ErrCodeInvalidRoleException, "encountered an error because your IAM role") {
				return retry.RetryableError(err)
			}

			return retry.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		output, err = conn.CreateFleetWithContext(ctx, input)
	}
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating Appstream Fleet (%s): %s", d.Get(names.AttrName).(string), err)
	}

	d.SetId(aws.StringValue(output.Fleet.Name))

	// Start fleet workflow
	_, err = conn.StartFleetWithContext(ctx, &appstream.StartFleetInput{
		Name: aws.String(d.Id()),
	})
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "starting Appstream Fleet (%s): %s", d.Id(), err)
	}

	if _, err = waitFleetStateRunning(ctx, conn, d.Id()); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for Appstream Fleet (%s) to be running: %s", d.Id(), err)
	}

	return append(diags, resourceFleetRead(ctx, d, meta)...)
}

func resourceFleetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppStreamConn(ctx)

	resp, err := conn.DescribeFleetsWithContext(ctx, &appstream.DescribeFleetsInput{Names: []*string{aws.String(d.Id())}})

	if !d.IsNewResource() && tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
		log.Printf("[WARN] Appstream Fleet (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading Appstream Fleet (%s): %s", d.Id(), err)
	}

	if len(resp.Fleets) == 0 {
		return sdkdiag.AppendErrorf(diags, "reading Appstream Fleet (%s): %s", d.Id(), "empty response")
	}

	if len(resp.Fleets) > 1 {
		return sdkdiag.AppendErrorf(diags, "reading Appstream Fleet (%s): %s", d.Id(), "multiple fleets found")
	}

	fleet := resp.Fleets[0]

	d.Set(names.AttrARN, fleet.Arn)

	if fleet.ComputeCapacityStatus != nil {
		if err = d.Set("compute_capacity", []interface{}{flattenComputeCapacity(fleet.ComputeCapacityStatus)}); err != nil {
			return create.AppendDiagSettingError(diags, names.AppStream, "Fleet", d.Id(), "compute_capacity", err)
		}
	} else {
		d.Set("compute_capacity", nil)
	}

	d.Set("created_time", aws.TimeValue(fleet.CreatedTime).Format(time.RFC3339))
	d.Set(names.AttrDescription, fleet.Description)
	d.Set(names.AttrDisplayName, fleet.DisplayName)
	d.Set("disconnect_timeout_in_seconds", fleet.DisconnectTimeoutInSeconds)

	if fleet.DomainJoinInfo != nil {
		if err = d.Set("domain_join_info", []interface{}{flattenDomainInfo(fleet.DomainJoinInfo)}); err != nil {
			return create.AppendDiagSettingError(diags, names.AppStream, "Fleet", d.Id(), "domain_join_info", err)
		}
	} else {
		d.Set("domain_join_info", nil)
	}

	d.Set("idle_disconnect_timeout_in_seconds", fleet.IdleDisconnectTimeoutInSeconds)
	d.Set("enable_default_internet_access", fleet.EnableDefaultInternetAccess)
	d.Set("fleet_type", fleet.FleetType)
	d.Set("iam_role_arn", fleet.IamRoleArn)
	d.Set("image_name", fleet.ImageName)
	d.Set("image_arn", fleet.ImageArn)
	d.Set(names.AttrInstanceType, fleet.InstanceType)
	d.Set("max_sessions_per_instance", fleet.MaxSessionsPerInstance)
	d.Set("max_user_duration_in_seconds", fleet.MaxUserDurationInSeconds)
	d.Set(names.AttrName, fleet.Name)
	d.Set(names.AttrState, fleet.State)
	d.Set("stream_view", fleet.StreamView)

	if fleet.VpcConfig != nil {
		if err = d.Set(names.AttrVPCConfig, []interface{}{flattenVPCConfig(fleet.VpcConfig)}); err != nil {
			return create.AppendDiagSettingError(diags, names.AppStream, "Fleet", d.Id(), names.AttrVPCConfig, err)
		}
	} else {
		d.Set(names.AttrVPCConfig, nil)
	}

	return diags
}

func resourceFleetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppStreamConn(ctx)
	input := &appstream.UpdateFleetInput{
		Name: aws.String(d.Id()),
	}
	shouldStop := false

	if d.HasChanges(names.AttrDescription, "domain_join_info", "enable_default_internet_access", "iam_role_arn", names.AttrInstanceType, "max_user_duration_in_seconds", "stream_view", names.AttrVPCConfig) {
		shouldStop = true
	}

	// Stop fleet workflow if needed
	if shouldStop {
		_, err := conn.StopFleetWithContext(ctx, &appstream.StopFleetInput{
			Name: aws.String(d.Id()),
		})
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "stopping Appstream Fleet (%s): %s", d.Id(), err)
		}
		if _, err = waitFleetStateStopped(ctx, conn, d.Id()); err != nil {
			return sdkdiag.AppendErrorf(diags, "waiting for Appstream Fleet (%s) to be stopped: %s", d.Id(), err)
		}
	}

	if d.HasChange("compute_capacity") {
		input.ComputeCapacity = expandComputeCapacity(d.Get("compute_capacity").([]interface{}))
	}

	if d.HasChange(names.AttrDescription) {
		input.Description = aws.String(d.Get(names.AttrDescription).(string))
	}

	if d.HasChange("domain_join_info") {
		input.DomainJoinInfo = expandDomainJoinInfo(d.Get("domain_join_info").([]interface{}))
	}

	if d.HasChange("disconnect_timeout_in_seconds") {
		input.DisconnectTimeoutInSeconds = aws.Int64(int64(d.Get("disconnect_timeout_in_seconds").(int)))
	}

	if d.HasChange("enable_default_internet_access") {
		input.EnableDefaultInternetAccess = aws.Bool(d.Get("enable_default_internet_access").(bool))
	}

	if d.HasChange("idle_disconnect_timeout_in_seconds") {
		input.IdleDisconnectTimeoutInSeconds = aws.Int64(int64(d.Get("idle_disconnect_timeout_in_seconds").(int)))
	}

	if d.HasChange(names.AttrDisplayName) {
		input.DisplayName = aws.String(d.Get(names.AttrDisplayName).(string))
	}

	if d.HasChange("image_name") {
		input.ImageName = aws.String(d.Get("image_name").(string))
	}

	if d.HasChange("image_arn") {
		input.ImageArn = aws.String(d.Get("image_arn").(string))
	}

	if d.HasChange("iam_role_arn") {
		input.IamRoleArn = aws.String(d.Get("iam_role_arn").(string))
	}

	if d.HasChange("stream_view") {
		input.StreamView = aws.String(d.Get("stream_view").(string))
	}

	if d.HasChange(names.AttrInstanceType) {
		input.InstanceType = aws.String(d.Get(names.AttrInstanceType).(string))
	}

	if d.HasChange("max_sessions_per_instance") {
		input.MaxSessionsPerInstance = aws.Int64(int64(d.Get("max_sessions_per_instance").(int)))
	}

	if d.HasChange("max_user_duration_in_seconds") {
		input.MaxUserDurationInSeconds = aws.Int64(int64(d.Get("max_user_duration_in_seconds").(int)))
	}

	if d.HasChange(names.AttrVPCConfig) {
		input.VpcConfig = expandVPCConfig(d.Get(names.AttrVPCConfig).([]interface{}))
	}

	_, err := conn.UpdateFleetWithContext(ctx, input)
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "updating Appstream Fleet (%s): %s", d.Id(), err)
	}

	// Start fleet workflow if stopped
	if shouldStop {
		_, err = conn.StartFleetWithContext(ctx, &appstream.StartFleetInput{
			Name: aws.String(d.Id()),
		})
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "starting Appstream Fleet (%s): %s", d.Id(), err)
		}

		if _, err = waitFleetStateRunning(ctx, conn, d.Id()); err != nil {
			return sdkdiag.AppendErrorf(diags, "waiting for Appstream Fleet (%s) to be running: %s", d.Id(), err)
		}
	}

	return append(diags, resourceFleetRead(ctx, d, meta)...)
}

func resourceFleetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppStreamConn(ctx)

	// Stop fleet workflow
	log.Printf("[DEBUG] Stopping AppStream Fleet: (%s)", d.Id())
	_, err := conn.StopFleetWithContext(ctx, &appstream.StopFleetInput{
		Name: aws.String(d.Id()),
	})
	if tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
		return diags
	}
	if err != nil {
		return sdkdiag.AppendErrorf(diags, "stopping Appstream Fleet (%s): %s", d.Id(), err)
	}

	if _, err = waitFleetStateStopped(ctx, conn, d.Id()); err != nil {
		return sdkdiag.AppendErrorf(diags, "waiting for Appstream Fleet (%s) to be stopped: %s", d.Id(), err)
	}

	log.Printf("[DEBUG] Deleting AppStream Fleet: (%s)", d.Id())
	_, err = conn.DeleteFleetWithContext(ctx, &appstream.DeleteFleetInput{
		Name: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, appstream.ErrCodeResourceNotFoundException) {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting Appstream Fleet (%s): %s", d.Id(), err)
	}

	return diags
}

func resourceFleetCustDiff(_ context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if diff.HasChange("domain_join_info") {
		o, n := diff.GetChange("domain_join_info")

		if reflect.DeepEqual(expandDomainJoinInfo(o.([]interface{})), expandDomainJoinInfo(n.([]interface{}))) {
			return diff.Clear("domain_join_info")
		}
	}
	return nil
}

func expandComputeCapacity(tfList []interface{}) *appstream.ComputeCapacity {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &appstream.ComputeCapacity{}

	attr := tfList[0].(map[string]interface{})
	if v, ok := attr["desired_instances"]; ok && v != 0 {
		apiObject.DesiredInstances = aws.Int64(int64(v.(int)))
	}

	if v, ok := attr["desired_sessions"]; ok && v != 0 {
		apiObject.DesiredSessions = aws.Int64(int64(v.(int)))
	}

	if reflect.DeepEqual(&appstream.ComputeCapacity{}, apiObject) {
		return nil
	}

	return apiObject
}

func flattenComputeCapacity(apiObject *appstream.ComputeCapacityStatus) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.DesiredUserSessions; v != nil {
		tfMap["desired_sessions"] = aws.Int64Value(v)
	}

	// desiredInstances is always returned by the API but cannot be used in conjunction with desiredSessions
	if v := apiObject.Desired; v != nil && tfMap["desired_sessions"] == nil {
		tfMap["desired_instances"] = aws.Int64Value(v)
	}

	if v := apiObject.Available; v != nil {
		tfMap["available"] = aws.Int64Value(v)
	}

	if v := apiObject.InUse; v != nil {
		tfMap["in_use"] = aws.Int64Value(v)
	}

	if v := apiObject.Running; v != nil {
		tfMap["running"] = aws.Int64Value(v)
	}

	if reflect.DeepEqual(map[string]interface{}{}, tfMap) {
		return nil
	}

	return tfMap
}

func expandDomainJoinInfo(tfList []interface{}) *appstream.DomainJoinInfo {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &appstream.DomainJoinInfo{}

	tfMap, ok := tfList[0].(map[string]interface{})

	if !ok {
		return nil
	}

	if v, ok := tfMap["directory_name"]; ok && v != "" {
		apiObject.DirectoryName = aws.String(v.(string))
	}

	if v, ok := tfMap["organizational_unit_distinguished_name"]; ok && v != "" {
		apiObject.OrganizationalUnitDistinguishedName = aws.String(v.(string))
	}

	if reflect.DeepEqual(&appstream.DomainJoinInfo{}, apiObject) {
		return nil
	}

	return apiObject
}

func flattenDomainInfo(apiObject *appstream.DomainJoinInfo) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.DirectoryName; v != nil && aws.StringValue(v) != "" {
		tfMap["directory_name"] = aws.StringValue(v)
	}

	if v := apiObject.OrganizationalUnitDistinguishedName; v != nil && aws.StringValue(v) != "" {
		tfMap["organizational_unit_distinguished_name"] = aws.StringValue(v)
	}

	if reflect.DeepEqual(map[string]interface{}{}, tfMap) {
		return nil
	}

	return tfMap
}

func expandVPCConfig(tfList []interface{}) *appstream.VpcConfig {
	if len(tfList) == 0 {
		return nil
	}

	apiObject := &appstream.VpcConfig{}

	tfMap := tfList[0].(map[string]interface{})
	if v, ok := tfMap[names.AttrSecurityGroupIDs]; ok {
		apiObject.SecurityGroupIds = flex.ExpandStringList(v.([]interface{}))
	}

	if v, ok := tfMap[names.AttrSubnetIDs]; ok {
		apiObject.SubnetIds = flex.ExpandStringList(v.([]interface{}))
	}

	if reflect.DeepEqual(&appstream.VpcConfig{}, apiObject) {
		return nil
	}

	return apiObject
}

func flattenVPCConfig(apiObject *appstream.VpcConfig) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.SecurityGroupIds; v != nil {
		tfMap[names.AttrSecurityGroupIDs] = aws.StringValueSlice(v)
	}

	if v := apiObject.SubnetIds; v != nil {
		tfMap[names.AttrSubnetIDs] = aws.StringValueSlice(v)
	}

	if reflect.DeepEqual(map[string]interface{}{}, tfMap) {
		return nil
	}

	return tfMap
}
