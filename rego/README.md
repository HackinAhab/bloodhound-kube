# Property Flattening and the '__private' Prefix

The '__private' prefix is used to indicate properties that are not meant to be exposed directly in the graph, but are still needed for edge generation. This allows us to keep the original maps for edge generation while still flattening other properties for easier querying.