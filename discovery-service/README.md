# Discovery service

Kernel service responsible for keeping track of which services are
running on which hosts, plus lots of information about those running
services - including which endpoints they have and which version they are.

Remember that we can happily run more than once **instance** of a service
on the same box. Each instance gets its own random UUID to identify it,
and we use this to uniquely identify and address each instance.

To ensure the data is fresh and valid, the discovery service is responsible
for sending **heartbeats** to instances periodically, removing any
services that do not respond to heartbeats within a timely fashion.

### Summary of concepts

#### Instance

A running service. Has a unique identifier (which corresponds to its RabbitMQ
queue name) plus information about the service, such as the name and the
version. Entirely plausible to have more than one local instance of the
same service, potentially with different versions.

#### Service

A collection of endpoints, grouped into a single unit of deployable code.

#### Endpoint

A single feature that can be invoked, with a specific request and response
definition.


## Architecture

The beating heart of the service is the registry. This has two jobs:

 - keeps track of **locally** registered services, sends these heartbeats,
   and removes any that become unhealthy
 - keeps in sync with all the other discovery service instances to build
   up a full picture of _all services_ running on every host (in a region)
   such that it can't quickly answer questions on these

The design relies on ZooKeeper to maintain shared state in the region, establishing
watches to react to changes invoked by other discovery service instances.

The core of this is the `syncer` loop with region registry:

	for {
		instanceIds, _, watch, err := zk.ChildrenW(rootNode)
		if err != nil {
			log.Errorf("[Discovery] Failed to read children, so exiting: %v", err)
			break
		}

		if err := r.sync(instanceIds); err != nil {
			log.Errorf("[Discovery] Failed to sync instances, so exiting: %v", err)
			break
		}

		// @todo not entirely sure what happens when zk conn.Close() happens - hopefully sender closes channel
		e := <-watch
		log.Debugf("[Discovery] Watch triggered for event %v", e)
	}

And then when we trigger a watch, we simply reload all children, watch again, and
then go and compare the child nodes with our in-memory structure:

	seen := map[string]bool{}
	for _, id := range instanceIds {
		log.Debugf("[Discovery] Attempting to sync %v", id)
		// do we know about this? assumption: registry never changes for an instance ID
		if _, ok := r.instances[id]; !ok {
			// look up and add this instance
			b, _, err := zk.Get(zkPathForInstance(id))
			if err != nil {
				return err
			}
			// unmarshal the document
			instance := &Instance{}
			if err := json.Unmarshal(b, instance); err != nil {
				return err
			}
			r.instances[id] = instance
		}
		seen[id] = true
	}

The key thing here is that we only go to ZK when we absolutely have to, in
order to `Get` the details of a new node we haven't yet seen.

For querying, everything is in-memory and uses a handy DSL:

	instances := registry.AllInstances().Filter(
		registry.MatchingServicePrefix("com.HailoOSS.kernel")
		)
	for _, i := range instances {
		log.Debugf("Instance: %v", i)
	}

