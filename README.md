# TaskEngine
*Is planned to be an easy and approachable solution for application integration*

TaskEngine consists of a simple model consisting of **Event Providers** and **Action Providers** these components are then chained together using a very simple description language.

**Example of the "language" layout**
```
<provider> <resource_title> {
  <resource_property_name>: <resource_property_value>
  <resource_property_name>:[
    <resource_property_array_value>
    <resource_property_array_value>
  ]
  <resource_property_name>:{
    <resource_property_map_name>:<resource_property_map_value>
    <resource_property_map_name>:<resource_property_map_value>
  }
}
```
*All values are interpreted as raw values, no single-quotes or double-quotes are needed*


#Event Providers

* **listener_event**

#Action Providers

* **listener_action** *(requires listener_event as it uses the http.ResponseWriter and http.Request from listener_event)*
* **localexec_action**
* **local_powershell_action**
* **mongo_find_action**

