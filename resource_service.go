package main

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/sky-uk/gonsx"
	"github.com/sky-uk/gonsx/api/service"
	"log"
)

func getSingleService(scopeid, name string, nsxclient *gonsx.NSXClient) (*service.ApplicationService, error) {
	getAllAPI := service.NewGetAll(scopeid)
	err := nsxclient.Do(getAllAPI)

	if err != nil {
		return nil, err
	}

	if getAllAPI.StatusCode() != 200 {
		return nil, fmt.Errorf("Status code: %d, Response: %s", getAllAPI.StatusCode(), getAllAPI.ResponseObject())
	}

	service := getAllAPI.GetResponse().FilterByName(name)

	if service.ObjectID == "" {
		return nil, fmt.Errorf("Not found %s", name)
	}

	return service, nil
}

func resourceService() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceCreate,
		Read:   resourceServiceRead,
		Delete: resourceServiceDelete,
		Update: resourceServiceUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"scopeid": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Required: true,
			},

			"protocol": {
				Type:     schema.TypeString,
				Required: true,
			},

			"ports": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceServiceCreate(d *schema.ResourceData, meta interface{}) error {
	nsxclient := meta.(*gonsx.NSXClient)
	var name, scopeid, description, protocol, ports string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	if v, ok := d.GetOk("scopeid"); ok {
		scopeid = v.(string)
	} else {
		return fmt.Errorf("scopeid argument is required")
	}

	if v, ok := d.GetOk("description"); ok {
		description = v.(string)
	} else {
		return fmt.Errorf("description argument is required")
	}

	if v, ok := d.GetOk("protocol"); ok {
		protocol = v.(string)
	} else {
		return fmt.Errorf("protocol argument is required")
	}
	ports = d.Get("ports").(string)

	// Create the API, use it and check for errors.
	log.Printf("[DEBUG] service.NewCreate(%s, %s, %s, %s, %s)", scopeid, name, description, protocol, ports)
	createAPI := service.NewCreate(scopeid, name, description, protocol, ports)
	err := nsxclient.Do(createAPI)

	if err != nil {
		return fmt.Errorf("Error: %v", err)
	}

	if createAPI.StatusCode() != 201 {
		return fmt.Errorf("%s", createAPI.ResponseObject())
	}

	// If we get here, everything is OK.  Set the ID for the Terraform state
	// and return the response from the READ method.
	d.SetId(createAPI.GetResponse())
	return resourceServiceRead(d, meta)
}

func resourceServiceRead(d *schema.ResourceData, meta interface{}) error {
	nsxclient := meta.(*gonsx.NSXClient)
	var scopeid, name string

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("scopeid"); ok {
		scopeid = v.(string)
	} else {
		return fmt.Errorf("scopeid argument is required")
	}

	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("name argument is required")
	}

	// Gather all the resources that are associated with the specified
	// scopeid.
	log.Printf("[DEBUG] service.NewGetAll(%s)", scopeid)
	api := service.NewGetAll(scopeid)
	err := nsxclient.Do(api)

	if err != nil {
		return err
	}

	// See if we can find our specifically named resource within the list of
	// resources associated with the scopeid.
	log.Printf("[DEBUG] api.GetResponse().FilterByName(\"%s\").ObjectID", name)
	serviceObject, err := getSingleService(scopeid, name, nsxclient) //nolint, maybe a reason to not trigger error
	id := serviceObject.ObjectID
	log.Printf("[DEBUG] id := %s", id)

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
	}

	return nil
}

func resourceServiceDelete(d *schema.ResourceData, meta interface{}) error {
	nsxclient := meta.(*gonsx.NSXClient)
	deleteAPI := service.NewDelete(d.Id())
	err := nsxclient.Do(deleteAPI)

	if err != nil {
		return err
	}
	log.Printf("[DEBUG] id %s deleted.", d.Id())
	return nil
}

func resourceServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	nsxclient := meta.(*gonsx.NSXClient)
	var scopeid string
	hasChanges := false

	// Gather the attributes for the resource.
	if v, ok := d.GetOk("scopeid"); ok {
		scopeid = v.(string)
	} else {
		return fmt.Errorf("scopeid argument is required")
	}

	// Do a GetAll on service resources that are associated with the specified scopeid.
	log.Printf("[DEBUG] service.NewGetAll(%s)", scopeid)
	api := service.NewGetAll(scopeid)
	err := nsxclient.Do(api)
	if err != nil {
		log.Printf("[DEBUG] Error during getting all service resources: %s", err)
		return err
	}

	// Find the resource with current name within all the scopeid resources.
	oldName, newName := d.GetChange("name")
	log.Printf("[DEBUG] api.GetResponse().FilterByName(\"%s\").ObjectID", oldName.(string))
	serviceObject, err := getSingleService(scopeid, oldName.(string), nsxclient) //nolint, maybe a reason to not trigger error
	id := serviceObject.ObjectID
	log.Printf("[DEBUG] id := %s", id)

	// If the resource has been removed manually, notify Terraform of this fact.
	if id == "" {
		d.SetId("")
		log.Printf("[DEBUG] Could not find the service resource with %s name, state will be cleared", oldName)
		return nil
	}

	if d.HasChange("name") {
		hasChanges = true
		serviceObject.Name = newName.(string)
		log.Printf("[DEBUG] Changing name of service from %s to %s", oldName.(string), newName.(string))
	}

	if d.HasChange("description") {
		hasChanges = true
		oldDesc, newDesc := d.GetChange("description")
		serviceObject.Description = newDesc.(string)
		log.Printf("[DEBUG] Changing description of service from %s to %s", oldDesc.(string), newDesc.(string))
	}

	if d.HasChange("protocol") || d.HasChange("ports") {
		hasChanges = true
		oldProtocol, newProtocol := d.GetChange("protocol")
		oldPorts, newPorts := d.GetChange("ports")
		newElement := service.Element{ApplicationProtocol: newProtocol.(string), Value: newPorts.(string)}
		serviceObject.Element = []service.Element{newElement}
		log.Printf("[DEBUG] Changing protocol and/or ports of service from %s:%s to %s:%s",
			oldProtocol.(string), oldPorts.(string), newProtocol.(string), newPorts.(string))
	}

	if hasChanges {
		updateAPI := service.NewUpdate(id, serviceObject)
		err = nsxclient.Do(updateAPI)

		if err != nil {
			log.Printf("[DEBUG] Error updating service resource: %s", err)
			return err
		}
	}
	return resourceServiceRead(d, meta)
}
