package newrelic

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/newrelic/newrelic-client-go/pkg/common"
	"github.com/newrelic/newrelic-client-go/pkg/entities"
	"github.com/newrelic/newrelic-client-go/pkg/errors"
	"github.com/newrelic/newrelic-client-go/pkg/synthetics"
)

func resourceNewRelicSyntheticsBrokenLinksMonitor() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNewRelicSyntheticsBrokenLinksMonitorCreate,
		ReadContext:   resourceNewRelicSyntheticsBrokenLinksMonitorRead,
		UpdateContext: resourceNewRelicSyntheticsBrokenLinksMonitorUpdate,
		DeleteContext: resourceNewRelicSyntheticsBrokenLinksMonitorDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: mergeSchemas(
			syntheticsBrokenLinksMonitorSchema(),
			syntheticsMonitorCommonSchema(),
			syntheticsMonitorLocationsAsStringsSchema(),
		),
	}
}

func syntheticsBrokenLinksMonitorSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"uri": {
			Type:        schema.TypeString,
			Description: "The URI the monitor runs against.",
			Required:    true,
		},
	}
}

func resourceNewRelicSyntheticsBrokenLinksMonitorCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	accountID := selectAccountID(providerConfig, d)

	monitorInput := buildSyntheticsBrokenLinksMonitorCreateInput(d)
	resp, err := client.Synthetics.SyntheticsCreateBrokenLinksMonitorWithContext(ctx, accountID, *monitorInput)
	if err != nil {
		return diag.FromErr(err)
	}

	errors := buildCreateSyntheticsMonitorResponseErrors(resp.Errors)
	if len(errors) > 0 {
		return errors
	}

	// Set attributes
	d.SetId(string(resp.Monitor.GUID))
	_ = d.Set("account_id", accountID)
	err = setSyntheticsMonitorAttributes(d, map[string]string{
		"guid":   string(resp.Monitor.GUID),
		"name":   resp.Monitor.Name,
		"period": string(resp.Monitor.Period),
		"status": string(resp.Monitor.Status),
		"uri":    resp.Monitor.Uri,
	})

	return diag.FromErr(err)
}

func resourceNewRelicSyntheticsBrokenLinksMonitorRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	accountID := selectAccountID(providerConfig, d)

	log.Printf("[INFO] Reading New Relic Synthetics monitor %s", d.Id())

	resp, err := client.Entities.GetEntityWithContext(ctx, common.EntityGUID(d.Id()))
	if err != nil {
		if _, ok := err.(*errors.NotFound); ok {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	switch e := (*resp).(type) {
	case *entities.SyntheticMonitorEntity:
		entity := (*resp).(*entities.SyntheticMonitorEntity)

		d.SetId(string(e.GUID))
		_ = d.Set("account_id", accountID)
		_ = d.Set("locations_public", getPublicLocationsFromEntityTags(entity.GetTags()))

		err = setSyntheticsMonitorAttributes(d, map[string]string{
			"guid":   string(e.GUID),
			"name":   entity.Name,
			"period": string(syntheticsMonitorPeriodValueMap[int(entity.GetPeriod())]),
			"status": string(entity.MonitorSummary.Status),
			"uri":    entity.MonitoredURL,
		})
	}

	return diag.FromErr(err)
}

func resourceNewRelicSyntheticsBrokenLinksMonitorUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerConfig := meta.(*ProviderConfig)
	client := providerConfig.NewClient
	guid := synthetics.EntityGUID(d.Id())

	monitorInput := buildSyntheticsBrokenLinksMonitorUpdateInput(d)
	resp, err := client.Synthetics.SyntheticsUpdateBrokenLinksMonitorWithContext(ctx, guid, *monitorInput)
	if err != nil {
		return diag.FromErr(err)
	}

	errors := buildUpdateSyntheticsMonitorResponseErrors(resp.Errors)
	if len(errors) > 0 {
		return errors
	}

	err = setSyntheticsMonitorAttributes(d, map[string]string{
		"guid":   string(resp.Monitor.GUID),
		"name":   resp.Monitor.Name,
		"period": string(resp.Monitor.Period),
		"status": string(resp.Monitor.Status),
		"uri":    resp.Monitor.Uri,
	})

	return diag.FromErr(err)
}

// NOTE: We can make rename this to reusable function for all new monitor types,
//
//	but the legacy function already has a good generic name (`resourceNewRelicSyntheticsMonitorDelete()`)
func resourceNewRelicSyntheticsBrokenLinksMonitorDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).NewClient
	guid := synthetics.EntityGUID(d.Id())

	log.Printf("[INFO] Deleting New Relic Synthetics monitor %s", d.Id())

	_, err := client.Synthetics.SyntheticsDeleteMonitorWithContext(ctx, guid)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diag.FromErr(err)
}
