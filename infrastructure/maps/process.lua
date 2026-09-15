-- Project-owned schema. No shapefiles or remote style resources are required.
node_keys = {"place", "amenity", "shop", "tourism", "addr:housenumber"}
function name_attribute()
 local name = Find("name:ru")
 if name == "" then name = Find("name") end
 if name ~= "" then Attribute("name",name) end
end
function node_function()
 if Find("place") ~= "" then
  Layer("place",false); name_attribute(); Attribute("class",Find("place"))
  if Find("place") == "city" then MinZoom(5) else MinZoom(10) end
 end
 if Find("addr:housenumber") ~= "" then
  Layer("housenumber",false); Attribute("number",Find("addr:housenumber"))
 end
 if Find("amenity") ~= "" or Find("shop") ~= "" or Find("tourism") ~= "" then
  Layer("poi",false); name_attribute()
 end
end
function way_function()
 local highway=Find("highway")
 if highway ~= "" then
  Layer("road",false); Attribute("class",highway); name_attribute()
  if highway ~= "motorway" and highway ~= "trunk" and highway ~= "primary" then MinZoom(11) end
 end
 if IsClosed() then
  if Find("building") ~= "" and Find("building") ~= "no" then
   Layer("building",true)
  end
  if Find("addr:housenumber") ~= "" then
   LayerAsCentroid("housenumber"); Attribute("number",Find("addr:housenumber"))
  end
  if Find("natural") == "water" or Find("waterway") == "riverbank" then Layer("water",true) end
  if Find("landuse") ~= "" or Find("leisure") == "park" then Layer("landuse",true) end
  if Find("amenity") ~= "" or Find("shop") ~= "" or Find("tourism") ~= "" then LayerAsCentroid("poi"); name_attribute() end
 end
end
