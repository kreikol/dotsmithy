// Paquete manifest: parseo y validación del manifiesto dots.yaml, el único
// contrato entre el motor y el contenido.
//
// Responsabilidad (F1): leer dots.yaml, validarlo y exponer un modelo tipado
// (capas, paquetes, hooks, externals, system) al resto del motor. Lógica pura
// y muy testeable. En F0 es solo el hueco.
package manifest
