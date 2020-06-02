Control plane
=============

Responsible for knowing how to route Hailo traffic, with two features:

 1. Which region to route to ("app pinning" feature)
 2. Which backend to send traffic to (H1, H2, load shedding/throttling)

Example usage:

    control := controlplane.New()

    // and then for each request we can yield a router
    router := control.Router(httpReq)

    // and then use the router to efficiently route according to backend/region
    region := router.Region()
    rule := router.Route()
