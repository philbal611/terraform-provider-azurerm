package azurerm

import (
	"fmt"
	"log"
	
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2018-10-01/consumption"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
	"github.com/satori/go.uuid"
)

func resourceArmConsumptionBudget() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmConsumptionBudgetCreate,
		Read:   resourceArmConsumptionBudgetRead,
		Delete: resourceArmConsumptionBudgetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"category": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(consumption.Cost),
					string(consumption.Usage),
				}, true),
			}

			"amount": {
				Type:     schema.TypeFloat,
				Required: true,
				ForceNew: true,
			},

			"time_grain": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					string(consumption.TimeGrainTypeAnnually),
					string(consumption.TimeGrainTypeMonthly),
					string(consumption.TimeGrainTypeQuarterly),
				}, true),
				DiffSuppressFunc: suppress.CaseDifference,
			},

			"filters": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"meters": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"resource_group_names": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceGroupNameSchema(),
						},
						"resource_ids": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: 	  &schema.Schema{
								Type: 		  schema.TypeString,
								ValidateFunc: azure.ValidateResourceID,
							},
						},
						"tags": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},

			"notification": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 5,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"threshold": {
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: validation.IntBetween(0, 1000),
						},
						"operator": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(consumption.EqualTo),
								string(consumption.GreaterThan),
								string(consumption.GreaterThanOrEqualTo),
							}, true),
						},
						"action_groups": {
							Type:     schema.TypeList,
							Optional: true;
							Elem: &schema.Schema{Type: schema.TypeString},
						},
						"contact_emails": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{Type: schema.TypeString},
						},
					}
				},
			},
		},
	}
}

func resourceArmConsumptionBudgetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).budgetClient
	ctx := meta.(*ArmClient).StopContext

	log.Printf("[INFO] preparing arguments for Azure ARM Budget creation.")

	name := d.Get("name").(string)
	category := d.Get("category").(string)
	amount := d.Get("amount").(string)
	timeGrain := d.Get("time_grain").(string)

	properties := consumption.BudgetProperties{
		Category: &category,
		Amount: &amount,
		TimeGrain: &timeGrain,
	}

	if _, ok := d.GetOk("filters"); ok {
		filters, err := expandAzureRmConsumptionBudgetFilters(d)
		if err != nil {
			return err
		}
		properties.Filters = filters
	}

	if _, ok := d.GetOk("notification"); ok {
		notifications, err := expandAzureRmConsumptionBudgetNotifications(d)
		if err != nil {
			return err
		}
		properties.Notifications = notifications
	}
}




func expandAzureRmConsumptionBudgetFilters(d *Schema.ResourceData) ([]consumption.Filters, error) {
	filtersConfig := d.Get("filters").(*schema.Set).List()
	filterConfig := filtersConfig[0].(map[string]interface{})
	filter := &consumption.Filters

	if r, ok := filterConfig["meters"].(*schema.Set); ok && r.Len() > 0 {
		var meters []uuid.UUID
		for _, v := range r.List() {
			s := v.(uuid.UUID)
			meters = append(meters, s)
		}
		filter.Meters = meters
	}

	if r, ok := filterConfig["resource_group_names"].(*schema.Set); ok && r.Len() > 0 {
		var rgNames []string
		for _, v := range r.List() {
			s := v.(string)
			rgNames = append(rgNames, s)
		}
		filter.ResourceGroups = rgNames
	}

	if r, ok := filterConfig["resource_ids"].(*schema.Set); ok && r.Len() > 0 {
		var resourceIds []string
		for _, v := range r.List() {
			s := v.(string)
			resourceIds = append(resourceIds, s)
		}
		filter.resourceIds = resourceIds
	}

	return filter, nil
}

func expandAzureRmConsumptionBudgetNotifications(d *schema.ResourceData) ([]consumption.Notification, error) {
	notificationConfigs := d.Get("notification").([]interface{})
	managed_notifications := make([]consumption.Notification, 0, len(notificationConfigs))

	for _, notificationConfig := range notificationConfigs {
		config := notificationConfig.(map[string]interface{})

		threshold := int32(config["threshold"].(int))
		operator := config["operator"].(string)

		properties := consumption.Notification{
			Threshold: &threshold,
			Operator: &operator,
		}

		if r, ok := config["contact_emails"].(*schema.Set); ok && r.Len() > 0 {
			var contactEmails []string
			for _, v := range r.List() {
				s := v.(string)
				contactEmails = append(contactEmails, s)
			}
			properties.ContactEmails = &contactEmails
		}

		if r, ok := config["action_groups"].(*schema.Set); ok && r.Len() > 0 {
			var actionGroups []string
			for _, v := range r.List() {
				s := v.(string)
				actionGroups = append(actionGroups, s)
			}
			properties.ContactGroups = &actionGroups
		}

		managed_notifications = append(managed_notifications, properties)
	}

	return managed_notifications, nil
}



func flattenAzureRmConsumptionBudgetNotifications(notifications *[]consumption.Notification) []map[string]interface{} {
	if notifications == nil {
		return []interface{}{}
	}

	result := make([]map[string]interface{}, 0)
	for _, notification := range notifications {
		notificationConfig := make(map[string]interface{})
		notificationConfig["threshold"] = *notification.Threshold
		notificationConfig["operator"] = *notification.Operator

		if emails := notificationConfig["contact_emails"]; emails != nil {
			notificationConfig["contact_emails"] = sliceToSet(*notification.ContactEmails)
		}
		if actionGroups := notificationConfig["action_groups"]; actionGroups != nil {
			notificationConfig["action_groups"] = sliceToSet(*notification.ContactGroups)
		}

		result = append(result, notificationConfig)
	}

	return result
}

func sliceToSet(slice []string) *schema.Set {
	set := &schema.Set{F: schema.HashString}
	for _, v := range slice {
		set.Add(v)
	}
	return set
}

func validateFilter(filterConfig map[string]interface{}, )