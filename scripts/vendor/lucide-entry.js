import { createIcons, ArrowLeftRight, ArrowRight, Bed, Building2, Calendar, CalendarX, ClipboardList, Clock, CreditCard, Heart, MapPin, Plane, Search, Star, TriangleAlert, Users } from 'lucide';
const icons = { ArrowLeftRight, ArrowRight, Bed, Building2, Calendar, CalendarX, ClipboardList, Clock, CreditCard, Heart, MapPin, Plane, Search, Star, TriangleAlert, Users };
// API sama dengan UMD lucide: lucide.createIcons() / lucide.createIcons({ root }).
window.lucide = { createIcons: (opts = {}) => createIcons({ ...opts, icons }) };
