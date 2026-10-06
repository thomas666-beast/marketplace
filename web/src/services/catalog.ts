import { api } from './api'

export interface Category {
  id: string
  parent_id: string | null
  slug: string
  name: string
  sort_order: number
  children?: Category[]
}

export interface Product {
  id: string
  seller_id: string
  category_id: string
  slug: string
  name: string
  description: string
  price_cents: number
  currency: 'RUB' | 'USD' | 'EUR'
  stock_quantity: number
  status: 'draft' | 'active' | 'archived'
  created_at: string
  updated_at: string
}

export interface ProductPage {
  items: Product[]
  next_cursor?: string
}

export interface ListProductsParams {
  category_id?: string
  q?: string
  cursor?: string
  limit?: number
}

export async function listCategories(): Promise<Category[]> {
  const { data } = await api.get<{ items: Category[] }>('/v1/categories')
  return data.items
}

export async function listProducts(params: ListProductsParams = {}): Promise<ProductPage> {
  const { data } = await api.get<ProductPage>('/v1/products', { params })
  return data
}

export async function getProduct(slug: string): Promise<Product> {
  const { data } = await api.get<Product>(`/v1/products/${slug}`)
  return data
}
