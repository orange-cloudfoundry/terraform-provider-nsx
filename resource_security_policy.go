package main

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sky-uk/gonsx"
	"github.com/sky-uk/gonsx/api/securitypolicy"
	"log"
)

func getSingleSecurityPolicy(name string, nsxclient *gonsx.NSXClient) (*securitypolicy.SecurityPolicy, error) {
	getAllAPI := securitypolicy.NewGetAll()
	err := nsxclient.Do(getAllAPI)

	if err != nil {
		return nil, err
	}

	if getAllAPI.StatusCode() != 200 {
		return nil, fmt.Errorf("Status code: %d, Response: %s", getAllAPI.StatusCode(), getAllAPI.ResponseObject())
	}

	log.Printf(fmt.Sprintf("[DEBUG] getAllAPI.GetResponse().FilterByName(\"%s\").ObjectID", name))
	securityPolicy := getAllAPI.GetResponse().FilterByName(name)


	return securityPolicy, nil
}

func resourceSecurityPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecurityPolicyCreate,
		Read:   resourceSecurityPolicyRead,
		Delete: resourceSecurityPolicyDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"precedence": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"securitygroups": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceSecurityPolicyCreate(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name, description, precedence string
	var securitygroups []string
	var actions []securitypolicy.Action

	// Gather the attributes for the resource.

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	if v, ok := d.GetOk("precedence"); ok {
		precedence = v.(string)
	} else {
		return fmt.Errorf("precedence argument is required")
	}

	if v, ok := d.GetOk("description"); ok {
		description = v.(string)
	} else {
		return fmt.Errorf("description argument is required")
	}

	if v, ok := d.GetOk("securitygroups"); ok {
		list := v.([]interface{})

		securitygroups = make([]string, len(list))
		for i, value := range list {
			groupID, ok := value.(string)
			if !ok {
				return fmt.Errorf("empty element found in securitygroups")
			}
			securitygroups[i] = groupID
		}
	} else {
		return fmt.Errorf("security groups argument is required")
	}

	log.Printf(fmt.Sprintf("[DEBUG] securitypolicy.NewCreate(%s, %s, %s, %s, %s)", name, precedence, description, securitygroups, actions))
	createAPI := securitypolicy.NewCreate(name, precedence, description, securitygroups, actions)
	err := nsxclient.Do(createAPI)

	if err != nil {
		return fmt.Errorf("Error creating security policy: %v", err)
	}

	if createAPI.StatusCode() != 201 {
		return fmt.Errorf("%s", createAPI.ResponseObject())
	}

	d.SetId(createAPI.GetResponse())
	return resourceSecurityPolicyRead(d, m)
}

func resourceSecurityPolicyRead(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name string

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	securityPolicyObject, err := getSingleSecurityPolicy(name, nsxclient)
	if err != nil {
		return err
	}
	id := securityPolicyObject.ObjectID
	log.Printf(fmt.Sprintf("[DEBUG] id := %s", id))

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
	}
	return nil
}

func resourceSecurityPolicyDelete(d *schema.ResourceData, m interface{}) error {
	nsxclient := m.(*gonsx.NSXClient)
	var name string

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	securityPolicyObject, err := getSingleSecurityPolicy(name, nsxclient)
	if err != nil {
		return err
	}
	id := securityPolicyObject.ObjectID
	log.Printf(fmt.Sprintf("[DEBUG] id := %s", id))

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
		return nil
	}

	// If we got here, the resource exists, so we attempt to delete it.
	// FIXME: we need to get terraform force call and pass it here.
	deleteAPI := securitypolicy.NewDelete(id, false)
	err = nsxclient.Do(deleteAPI)

	if err != nil {
		return err
	}

	// If we got here, the resource had existed, we deleted it and there was
	// no error.  Notify Terraform of this fact and return successful
	// completion.
	d.SetId("")
	log.Printf(fmt.Sprintf("[DEBUG] id %s deleted.", id))

	return nil
}
