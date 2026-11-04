import 'leaflet';

declare module 'leaflet' {
  type HeatLatLngTuple = [number, number, number];
  interface HeatLayer extends Layer {
    addTo(map: Map): this;
    remove(): this;
    setOptions(options: any): this;
    setLatLngs(latlngs: HeatLatLngTuple[] | LatLng[]): this;
  }
  function heatLayer(
    latlngs: HeatLatLngTuple[] | LatLng[],
    options?: any
  ): HeatLayer;
}
